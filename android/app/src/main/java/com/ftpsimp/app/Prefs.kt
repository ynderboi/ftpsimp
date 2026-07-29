package com.ftpsimp.app

import android.content.Context
import android.content.SharedPreferences
import android.net.Uri

class Prefs(context: Context) {
    private val sp: SharedPreferences =
        context.getSharedPreferences("ftpsimp", Context.MODE_PRIVATE)

    var port: Int
        get() = sp.getInt(KEY_PORT, 8080)
        set(value) = sp.edit().putInt(KEY_PORT, value).apply()

    var treeUri: String?
        get() = sp.getString(KEY_TREE, null)
        set(value) = sp.edit().putString(KEY_TREE, value).apply()

    var openNoAuth: Boolean
        get() = sp.getBoolean(KEY_OPEN, false)
        set(value) = sp.edit().putBoolean(KEY_OPEN, value).apply()

    var readOnly: Boolean
        get() = sp.getBoolean(KEY_RO, false)
        set(value) = sp.edit().putBoolean(KEY_RO, value).apply()

    var fixedPin: String?
        get() = sp.getString(KEY_PIN, null)?.ifBlank { null }
        set(value) = sp.edit().putString(KEY_PIN, value).apply()

    fun treeUriOrNull(): Uri? = treeUri?.let(Uri::parse)

    companion object {
        private const val KEY_PORT = "port"
        private const val KEY_TREE = "tree_uri"
        private const val KEY_OPEN = "open_no_auth"
        private const val KEY_RO = "read_only"
        private const val KEY_PIN = "fixed_pin"
    }
}
