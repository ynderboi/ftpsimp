package com.ftpsimp.app

import android.Manifest
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.widget.Toast
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.documentfile.provider.DocumentFile
import com.ftpsimp.app.databinding.ActivityMainBinding

class MainActivity : AppCompatActivity() {
    private lateinit var binding: ActivityMainBinding
    private lateinit var prefs: Prefs

    private val openTree = registerForActivityResult(
        ActivityResultContracts.OpenDocumentTree()
    ) { uri: Uri? ->
        if (uri == null) return@registerForActivityResult
        contentResolver.takePersistableUriPermission(
            uri,
            Intent.FLAG_GRANT_READ_URI_PERMISSION or Intent.FLAG_GRANT_WRITE_URI_PERMISSION
        )
        prefs.treeUri = uri.toString()
        updateFolderLabel()
        Toast.makeText(this, "Папка сохранена", Toast.LENGTH_SHORT).show()
        if (ServerService.isRunning) {
            ServerService.stop(this)
            ServerService.start(this)
        }
    }

    private val permissionLauncher = registerForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions()
    ) { startOrStop() }

    private val stateReceiver = object : BroadcastReceiver() {
        override fun onReceive(context: Context?, intent: Intent?) {
            if (intent?.action != ServerService.ACTION_STATE) return
            renderState(
                running = intent.getBooleanExtra(ServerService.EXTRA_RUNNING, false),
                error = intent.getStringExtra(ServerService.EXTRA_ERROR),
                root = intent.getStringExtra(ServerService.EXTRA_ROOT),
                urls = intent.getStringArrayListExtra(ServerService.EXTRA_URLS).orEmpty()
            )
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)
        prefs = Prefs(this)

        binding.btnPickFolder.setOnClickListener { openTree.launch(null) }
        binding.btnToggle.setOnClickListener { ensurePermsAndToggle() }

        updateFolderLabel()
        renderState(ServerService.isRunning, null, ServerService.currentRootLabel, emptyList())
        if (ServerService.isRunning) {
            val urls = NetworkUtils.ipv4Addresses(this)
                .map { "http://$it:${ServerService.currentPort}" }
            renderState(true, null, ServerService.currentRootLabel, urls)
        }
    }

    override fun onStart() {
        super.onStart()
        val filter = IntentFilter(ServerService.ACTION_STATE)
        ContextCompat.registerReceiver(
            this,
            stateReceiver,
            filter,
            ContextCompat.RECEIVER_NOT_EXPORTED
        )
    }

    override fun onStop() {
        unregisterReceiver(stateReceiver)
        super.onStop()
    }

    private fun ensurePermsAndToggle() {
        val needed = mutableListOf<String>()
        if (Build.VERSION.SDK_INT >= 33) {
            if (ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS)
                != PackageManager.PERMISSION_GRANTED
            ) {
                needed += Manifest.permission.POST_NOTIFICATIONS
            }
        }
        if (needed.isNotEmpty()) {
            permissionLauncher.launch(needed.toTypedArray())
        } else {
            startOrStop()
        }
    }

    private fun startOrStop() {
        if (ServerService.isRunning) {
            ServerService.stop(this)
        } else {
            ServerService.start(this)
        }
    }

    private fun updateFolderLabel() {
        val uri = prefs.treeUriOrNull()
        binding.folderPath.text = if (uri != null) {
            DocumentFile.fromTreeUri(this, uri)?.name
                ?: uri.lastPathSegment
                ?: uri.toString()
        } else {
            ServerService.defaultShareDir(this).absolutePath + " (по умолчанию)"
        }
    }

    private fun renderState(
        running: Boolean,
        error: String?,
        root: String?,
        urls: List<String>
    ) {
        if (!error.isNullOrBlank()) {
            Toast.makeText(this, error, Toast.LENGTH_LONG).show()
        }
        binding.status.text = if (running) {
            getString(R.string.server_running)
        } else {
            getString(R.string.server_stopped)
        }
        binding.btnToggle.text = if (running) {
            getString(R.string.stop_server)
        } else {
            getString(R.string.start_server)
        }
        if (!root.isNullOrBlank()) {
            binding.folderPath.text = root
        }
        binding.urls.text = when {
            running && urls.isNotEmpty() ->
                "Откройте на другом устройстве:\n" + urls.joinToString("\n")
            running -> "Нет Wi‑Fi адреса. Подключитесь к сети."
            else -> ""
        }
    }
}
