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

    fun treeUriOrNull(): Uri? = treeUri?.let(Uri::parse)

    companion object {
        private const val KEY_PORT = "port"
        private const val KEY_TREE = "tree_uri"
    }
}
