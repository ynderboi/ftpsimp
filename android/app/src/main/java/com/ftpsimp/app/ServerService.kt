package com.ftpsimp.app

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Build
import android.os.IBinder
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import androidx.documentfile.provider.DocumentFile
import java.io.File

class ServerService : Service() {
    private var server: FileServer? = null

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_STOP -> {
                stopServer()
                stopForeground(STOP_FOREGROUND_REMOVE)
                stopSelf()
                return START_NOT_STICKY
            }
            else -> startServer()
        }
        return START_STICKY
    }

    private fun startServer() {
        if (server != null) {
            broadcastState(true)
            return
        }
        val prefs = Prefs(this)
        val root = resolveRoot(prefs) ?: run {
            broadcastState(false, "Сначала выберите папку")
            stopSelf()
            return
        }
        createChannel()
        val authOn = !prefs.openNoAuth
        val pin = prefs.fixedPin ?: FileServer.generatePin()
        currentPort = prefs.port
        currentPin = if (authOn) pin else ""
        currentReadOnly = prefs.readOnly
        startForeground(NOTIF_ID, buildNotification(prefs.port))
        try {
            val s = FileServer(this, prefs.port, root, pin, authOn, prefs.readOnly)
            s.updateRoot(root)
            s.start()
            server = s
            isRunning = true
            currentRootLabel = s.rootLabel
            broadcastState(true)
        } catch (e: Exception) {
            currentPin = ""
            stopForeground(STOP_FOREGROUND_REMOVE)
            broadcastState(false, e.message ?: "Ошибка запуска")
            stopSelf()
        }
    }

    private fun stopServer() {
        try {
            server?.stop()
        } catch (_: Exception) {
        }
        server = null
        isRunning = false
        currentPin = ""
        broadcastState(false)
    }

    private fun resolveRoot(prefs: Prefs): Uri? {
        prefs.treeUriOrNull()?.let { uri ->
            val doc = DocumentFile.fromTreeUri(this, uri)
            if (doc != null && doc.canRead()) return uri
        }
        val dir = getExternalFilesDir(null) ?: filesDir
        if (!dir.exists()) dir.mkdirs()
        return Uri.fromFile(dir)
    }

    private fun broadcastState(running: Boolean, error: String? = null) {
        val urls = if (running) {
            NetworkUtils.ipv4Addresses(this).map { "http://$it:$currentPort" }
        } else {
            emptyList()
        }
        sendBroadcast(
            Intent(ACTION_STATE).setPackage(packageName)
                .putExtra(EXTRA_RUNNING, running)
                .putExtra(EXTRA_ERROR, error)
                .putExtra(EXTRA_ROOT, currentRootLabel)
                .putExtra(EXTRA_PIN, currentPin)
                .putExtra(EXTRA_READONLY, currentReadOnly)
                .putStringArrayListExtra(EXTRA_URLS, ArrayList(urls))
        )
    }

    private fun createChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val nm = getSystemService(NotificationManager::class.java)
        nm.createNotificationChannel(
            NotificationChannel(
                CHANNEL_ID,
                getString(R.string.notification_channel),
                NotificationManager.IMPORTANCE_LOW
            )
        )
    }

    private fun buildNotification(port: Int): Notification {
        val open = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )
        val stop = PendingIntent.getService(
            this,
            1,
            Intent(this, ServerService::class.java).setAction(ACTION_STOP),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )
        val ips = NetworkUtils.ipv4Addresses(this).joinToString(", ")
        val pinPart = if (currentPin.isNotEmpty()) " · PIN $currentPin" else ""
        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle(getString(R.string.server_running))
            .setContentText(
                if (ips.isNotEmpty()) "$ips:$port$pinPart"
                else getString(R.string.notification_text)
            )
            .setSmallIcon(R.drawable.ic_launcher)
            .setContentIntent(open)
            .addAction(0, getString(R.string.stop_server), stop)
            .setOngoing(true)
            .build()
    }

    override fun onDestroy() {
        stopServer()
        super.onDestroy()
    }

    companion object {
        const val ACTION_STOP = "com.ftpsimp.app.STOP"
        const val ACTION_STATE = "com.ftpsimp.app.STATE"
        const val EXTRA_RUNNING = "running"
        const val EXTRA_ERROR = "error"
        const val EXTRA_ROOT = "root"
        const val EXTRA_URLS = "urls"
        const val EXTRA_PIN = "pin"
        const val EXTRA_READONLY = "readonly"
        private const val CHANNEL_ID = "ftpsimp_server"
        private const val NOTIF_ID = 42

        @Volatile
        var isRunning: Boolean = false
            private set

        @Volatile
        var currentPort: Int = 8080
            private set

        @Volatile
        var currentRootLabel: String = ""
            private set

        @Volatile
        var currentPin: String = ""
            private set

        @Volatile
        var currentReadOnly: Boolean = false
            private set

        fun start(context: Context) {
            val i = Intent(context, ServerService::class.java)
            ContextCompat.startForegroundService(context, i)
        }

        fun stop(context: Context) {
            context.startService(
                Intent(context, ServerService::class.java).setAction(ACTION_STOP)
            )
        }

        fun defaultShareDir(context: Context): File {
            val dir = context.getExternalFilesDir(null) ?: context.filesDir
            if (!dir.exists()) dir.mkdirs()
            return dir
        }
    }
}
