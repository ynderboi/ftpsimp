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
import java.security.SecureRandom
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.TimeZone
import java.util.concurrent.ConcurrentHashMap

class FileServer(
    private val context: Context,
    port: Int,
    private var rootUri: Uri,
    private val pin: String,
    private val authOn: Boolean,
    private val readOnly: Boolean,
) : NanoHTTPD(port) {

    @Volatile
    var rootLabel: String = rootUri.toString()
        private set

    private val sessions = ConcurrentHashMap<String, Long>()
    private val rng = SecureRandom()

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
                uri == "/api/status" -> handleStatus(session)
                uri == "/api/login" && session.method == Method.POST -> handleLogin(session)
                uri == "/api/logout" && session.method == Method.POST -> handleLogout(session)
                uri.startsWith("/api/") && needsAuth(uri) && !authenticated(session) ->
                    text(Response.Status.UNAUTHORIZED, "unauthorized")
                uri.startsWith("/api/") && readOnly && isWrite(uri, session.method) ->
                    text(Response.Status.FORBIDDEN, "read-only mode")
                uri == "/api/list" -> handleList(session)
                uri == "/api/download" -> handleDownload(session)
                uri == "/api/upload" && session.method == Method.POST -> handleUpload(session)
                uri == "/api/mkdir" && session.method == Method.POST -> handleMkdir(session)
                uri == "/api/delete" && (session.method == Method.POST || session.method == Method.DELETE) ->
                    handleDelete(session)
                uri == "/api/info" -> json(
                    JSONObject().put("root", rootLabel).put("readOnly", readOnly)
                )
                uri == "/api/settings" && session.method == Method.GET -> json(
                    JSONObject()
                        .put("root", rootLabel)
                        .put("readOnly", readOnly)
                        .put("settingsLocal", true)
                        .put("canChangeRoot", false)
                )
                uri == "/api/settings" && session.method == Method.POST ->
                    text(Response.Status.FORBIDDEN, "На Android смените папку в приложении")
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

    private fun needsAuth(uri: String): Boolean {
        if (!authOn) return false
        return uri != "/api/status" && uri != "/api/login" && uri != "/api/logout"
    }

    private fun isWrite(uri: String, method: Method): Boolean {
        return when (uri) {
            "/api/upload", "/api/mkdir", "/api/delete" -> true
            "/api/settings" -> method == Method.POST
            else -> false
        }
    }

    private fun authenticated(session: IHTTPSession): Boolean {
        if (!authOn) return true
        val id = sessionId(session) ?: return false
        val exp = sessions[id] ?: return false
        if (System.currentTimeMillis() > exp) {
            sessions.remove(id)
            return false
        }
        return true
    }

    private fun sessionId(session: IHTTPSession): String? {
        val auth = session.headers["authorization"]
        if (!auth.isNullOrBlank() && auth.lowercase().startsWith("bearer ")) {
            return auth.substring(7).trim().ifBlank { null }
        }
        val hdr = session.headers["x-session-token"]?.trim()
        if (!hdr.isNullOrBlank()) return hdr
        val q = query(session, "token").trim()
        if (q.isNotEmpty()) return q
        return cookie(session, COOKIE)
    }

    private fun handleStatus(session: IHTTPSession): Response {
        val ok = authenticated(session)
        val obj = JSONObject()
            .put("authRequired", authOn)
            .put("authenticated", ok)
            .put("readOnly", readOnly)
            .put("settingsLocal", true)
        if (ok) obj.put("root", rootLabel)
        return json(obj)
    }

    private fun handleLogin(session: IHTTPSession): Response {
        if (!authOn) {
            return json(JSONObject().put("ok", true).put("authRequired", false).put("root", rootLabel))
        }
        val body = readJsonBody(session)
        val got = body.optString("pin").trim()
        if (got != pin) {
            Thread.sleep(200)
            return text(Response.Status.UNAUTHORIZED, "invalid pin")
        }
        val id = newSessionId()
        val exp = System.currentTimeMillis() + SESSION_TTL_MS
        sessions[id] = exp
        val res = json(
            JSONObject()
                .put("ok", true)
                .put("token", id)
                .put("root", rootLabel)
                .put("readOnly", readOnly)
        )
        res.addHeader("Set-Cookie", "$COOKIE=$id; Path=/; HttpOnly; SameSite=Lax; Max-Age=${SESSION_TTL_MS / 1000}")
        return res
    }

    private fun handleLogout(session: IHTTPSession): Response {
        sessionId(session)?.let { sessions.remove(it) }
        val res = json(JSONObject().put("ok", "1"))
        res.addHeader("Set-Cookie", "$COOKIE=; Path=/; HttpOnly; Max-Age=0")
        return res
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
        val ascii = name.filter { it.code in 32..126 && it != '"' && it != '\\' }.ifEmpty { "download" }
        val enc = java.net.URLEncoder.encode(name, "UTF-8").replace("+", "%20")
        res.addHeader("Content-Disposition", "attachment; filename=\"$ascii\"; filename*=UTF-8''$enc")
        return res
    }

    private fun handleUpload(session: IHTTPSession): Response {
        val rel = query(session, "path")
        val overwrite = query(session, "overwrite") == "1"
        val dir = resolveDir(rel) ?: return bad("target folder not found")
        val files = HashMap<String, String>()
        session.parseBody(files)
        val saved = JSONArray()
        for ((key, tmpPath) in files) {
            if (!key.startsWith("files")) continue
            val name = session.parameters[key]?.firstOrNull()
                ?: java.io.File(tmpPath).name
            val safe = name.substringAfterLast('/').substringAfterLast('\\')
            if (safe.isBlank() || safe == "." || safe == "..") return bad("invalid filename")
            val existing = dir.findFile(safe)
            if (existing != null && !overwrite) {
                return text(Response.Status.CONFLICT, "file exists: $safe (use overwrite)")
            }
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
        if (name.isEmpty() || name.contains('/') || name.contains('\\') || name == "." || name == "..") {
            return bad("invalid name")
        }
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

    private fun cookie(session: IHTTPSession, name: String): String? {
        val raw = session.headers["cookie"] ?: return null
        raw.split(';').forEach { part ->
            val kv = part.trim().split('=', limit = 2)
            if (kv.size == 2 && kv[0] == name) return kv[1]
        }
        return null
    }

    private fun newSessionId(): String {
        val b = ByteArray(32)
        rng.nextBytes(b)
        return b.joinToString("") { "%02x".format(it) }
    }

    private fun json(obj: JSONObject): Response {
        val bytes = obj.toString().toByteArray(StandardCharsets.UTF_8)
        val res = newFixedLengthResponse(
            Response.Status.OK,
            "application/json; charset=utf-8",
            ByteArrayInputStream(bytes),
            bytes.size.toLong()
        )
        res.addHeader("X-Content-Type-Options", "nosniff")
        return res
    }

    private fun text(status: Response.Status, msg: String): Response =
        newFixedLengthResponse(status, MIME_PLAINTEXT, msg)

    private fun bad(msg: String): Response =
        newFixedLengthResponse(Response.Status.BAD_REQUEST, MIME_PLAINTEXT, msg)

    companion object {
        private const val COOKIE = "ftpsimp_sess"
        private const val SESSION_TTL_MS = 24L * 60 * 60 * 1000

        fun generatePin(): String {
            val n = SecureRandom().nextInt(1_000_000)
            return String.format(Locale.US, "%06d", n)
        }

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
