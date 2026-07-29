package com.ftpsimp.app

import android.content.Context
import android.net.Uri
import android.webkit.MimeTypeMap
import androidx.documentfile.provider.DocumentFile
import fi.iki.elonen.NanoHTTPD
import org.json.JSONArray
import org.json.JSONObject
import java.io.ByteArrayInputStream
import java.io.FileNotFoundException
import java.net.URLDecoder
import java.nio.charset.StandardCharsets
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.TimeZone

class FileServer(
    private val context: Context,
    port: Int,
    private var rootUri: Uri,
) : NanoHTTPD(port) {

    @Volatile
    var rootLabel: String = rootUri.toString()
        private set

    fun updateRoot(uri: Uri) {
        rootUri = uri
        rootLabel = when (uri.scheme) {
            "file" -> uri.path ?: uri.toString()
            else -> DocumentFile.fromTreeUri(context, uri)?.name
                ?: uri.lastPathSegment
                ?: uri.toString()
        }
    }

    override fun serve(session: IHTTPSession): Response {
        val uri = session.uri.substringBefore('?')
        return try {
            when {
                uri == "/api/list" -> handleList(session)
                uri == "/api/download" -> handleDownload(session)
                uri == "/api/upload" && session.method == Method.POST -> handleUpload(session)
                uri == "/api/mkdir" && session.method == Method.POST -> handleMkdir(session)
                uri == "/api/delete" && (session.method == Method.POST || session.method == Method.DELETE) ->
                    handleDelete(session)
                uri == "/api/info" || uri == "/api/settings" && session.method == Method.GET ->
                    json(JSONObject().put("root", rootLabel))
                uri == "/api/settings" && session.method == Method.POST ->
                    json(JSONObject().put("root", rootLabel).put("note", "На Android смените папку в приложении"))
                uri == "/" || uri.isEmpty() -> asset("web/index.html", "text/html")
                uri.startsWith("/") -> {
                    val path = "web" + uri
                    val mime = mimeFromName(uri.substringAfterLast('/'))
                    asset(path, mime)
                }
                else -> newFixedLengthResponse(Response.Status.NOT_FOUND, MIME_PLAINTEXT, "not found")
            }
        } catch (e: Exception) {
            newFixedLengthResponse(
                Response.Status.INTERNAL_ERROR,
                MIME_PLAINTEXT,
                e.message ?: "error"
            )
        }
    }

    private fun handleList(session: IHTTPSession): Response {
        val rel = query(session, "path")
        val dir = resolveDir(rel) ?: return bad("not found")
        val entries = JSONArray()
        val files = dir.listFiles().orEmpty().sortedWith(
            compareBy<DocumentFile> { !it.isDirectory }.thenBy { it.name?.lowercase(Locale.ROOT) ?: "" }
        )
        for (f in files) {
            val name = f.name ?: continue
            val childRel = joinRel(rel, name)
            entries.put(
                JSONObject()
                    .put("name", name)
                    .put("path", childRel)
                    .put("isDir", f.isDirectory)
                    .put("size", if (f.isFile) f.length() else 0)
                    .put("modTime", iso(f.lastModified()))
            )
        }
        return json(
            JSONObject()
                .put("path", cleanRel(rel))
                .put("entries", entries)
        )
    }

    private fun handleDownload(session: IHTTPSession): Response {
        val rel = query(session, "path")
        val file = resolveFile(rel) ?: return bad("file not found")
        if (!file.isFile) return bad("file not found")
        val name = file.name ?: "file"
        val stream = context.contentResolver.openInputStream(file.uri)
            ?: return bad("cannot open")
        val mime = file.type ?: mimeFromName(name)
        val res = newFixedLengthResponse(Response.Status.OK, mime, stream, file.length())
        res.addHeader("Content-Disposition", "attachment; filename=\"$name\"")
        return res
    }

    private fun handleUpload(session: IHTTPSession): Response {
        val rel = query(session, "path")
        val dir = resolveDir(rel) ?: return bad("target folder not found")
        val files = HashMap<String, String>()
        session.parseBody(files)
        val saved = JSONArray()
        for ((key, tmpPath) in files) {
            if (!key.startsWith("files")) continue
            val name = session.parameters[key]?.firstOrNull()
                ?: java.io.File(tmpPath).name
            val safe = name.substringAfterLast('/').substringAfterLast('\\')
            val existing = dir.findFile(safe)
            existing?.delete()
            val target = dir.createFile(mimeFromName(safe), safe) ?: return bad("cannot create $safe")
            context.contentResolver.openOutputStream(target.uri)?.use { out ->
                java.io.File(tmpPath).inputStream().use { it.copyTo(out) }
            } ?: return bad("cannot write $safe")
            saved.put(safe)
        }
        if (saved.length() == 0) return bad("no files")
        return json(JSONObject().put("saved", saved))
    }

    private fun handleMkdir(session: IHTTPSession): Response {
        val body = readJsonBody(session)
        val rel = body.optString("path")
        val name = body.optString("name").trim()
        if (name.isEmpty() || name.contains('/') || name.contains('\\')) return bad("invalid name")
        val dir = resolveDir(rel) ?: return bad("not found")
        if (dir.findFile(name) != null) return bad("already exists")
        dir.createDirectory(name) ?: return bad("cannot create")
        return json(JSONObject().put("ok", "1"))
    }

    private fun handleDelete(session: IHTTPSession): Response {
        val body = readJsonBody(session)
        val rel = cleanRel(body.optString("path"))
        if (rel.isEmpty()) return bad("cannot delete root")
        val file = resolveAny(rel) ?: return bad("not found")
        if (!file.delete()) return bad("cannot delete")
        return json(JSONObject().put("ok", "1"))
    }

    private fun rootDoc(): DocumentFile? {
        return if (rootUri.scheme == "file") {
            val path = rootUri.path ?: return null
            DocumentFile.fromFile(java.io.File(path))
        } else {
            DocumentFile.fromTreeUri(context, rootUri)
        }
    }

    private fun resolveDir(rel: String): DocumentFile? {
        var cur = rootDoc() ?: return null
        for (part in cleanRel(rel).split('/').filter { it.isNotEmpty() }) {
            cur = cur.findFile(part) ?: return null
            if (!cur.isDirectory) return null
        }
        return cur
    }

    private fun resolveFile(rel: String): DocumentFile? {
        val f = resolveAny(rel) ?: return null
        return if (f.isFile) f else null
    }

    private fun resolveAny(rel: String): DocumentFile? {
        val parts = cleanRel(rel).split('/').filter { it.isNotEmpty() }
        if (parts.isEmpty()) return null
        var cur = rootDoc() ?: return null
        for (part in parts) {
            cur = cur.findFile(part) ?: return null
        }
        return cur
    }

    private fun asset(path: String, mime: String): Response {
        return try {
            val stream = context.assets.open(path)
            newChunkedResponse(Response.Status.OK, mime, stream)
        } catch (_: FileNotFoundException) {
            newFixedLengthResponse(Response.Status.NOT_FOUND, MIME_PLAINTEXT, "not found")
        }
    }

    private fun readJsonBody(session: IHTTPSession): JSONObject {
        val map = HashMap<String, String>()
        session.parseBody(map)
        val raw = map["postData"] ?: ""
        if (raw.isBlank()) return JSONObject()
        return JSONObject(raw)
    }

    private fun query(session: IHTTPSession, key: String): String {
        val v = session.parms[key] ?: return ""
        return URLDecoder.decode(v, StandardCharsets.UTF_8.name())
    }

    private fun json(obj: JSONObject): Response {
        val bytes = obj.toString().toByteArray(StandardCharsets.UTF_8)
        return newFixedLengthResponse(
            Response.Status.OK,
            "application/json; charset=utf-8",
            ByteArrayInputStream(bytes),
            bytes.size.toLong()
        )
    }

    private fun bad(msg: String): Response =
        newFixedLengthResponse(Response.Status.BAD_REQUEST, MIME_PLAINTEXT, msg)

    companion object {
        private fun cleanRel(rel: String): String {
            var r = rel.trim().replace('\\', '/')
            while (r.startsWith("/")) r = r.drop(1)
            if (r == "." || r == "..") return ""
            return r.split('/').filter { it.isNotEmpty() && it != "." && it != ".." }.joinToString("/")
        }

        private fun joinRel(parent: String, name: String): String {
            val p = cleanRel(parent)
            return if (p.isEmpty()) name else "$p/$name"
        }

        private fun mimeFromName(name: String): String {
            val ext = name.substringAfterLast('.', "").lowercase(Locale.ROOT)
            if (ext.isEmpty()) return "application/octet-stream"
            return MimeTypeMap.getSingleton().getMimeTypeFromExtension(ext)
                ?: when (ext) {
                    "js" -> "application/javascript"
                    "css" -> "text/css"
                    "html", "htm" -> "text/html"
                    "json" -> "application/json"
                    else -> "application/octet-stream"
                }
        }

        private fun iso(ms: Long): String {
            val fmt = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US)
            fmt.timeZone = TimeZone.getTimeZone("UTC")
            return fmt.format(Date(ms))
        }
    }
}
