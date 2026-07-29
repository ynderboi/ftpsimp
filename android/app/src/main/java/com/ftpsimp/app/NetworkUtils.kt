package com.ftpsimp.app

import android.content.Context
import android.net.ConnectivityManager
import android.net.LinkProperties
import android.net.NetworkCapabilities
import java.net.Inet4Address
import java.net.NetworkInterface

object NetworkUtils {
    fun ipv4Addresses(context: Context): List<String> {
        val fromCm = fromConnectivityManager(context)
        if (fromCm.isNotEmpty()) return fromCm
        return fromInterfaces()
    }

    private fun fromConnectivityManager(context: Context): List<String> {
        val cm = context.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
        val out = linkedSetOf<String>()
        for (network in cm.allNetworks) {
            val caps = cm.getNetworkCapabilities(network) ?: continue
            if (!caps.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) &&
                !caps.hasTransport(NetworkCapabilities.TRANSPORT_ETHERNET)
            ) {
                continue
            }
            val lp: LinkProperties = cm.getLinkProperties(network) ?: continue
            for (addr in lp.linkAddresses) {
                val ip = addr.address
                if (ip is Inet4Address && !ip.isLoopbackAddress) {
                    out += ip.hostAddress ?: continue
                }
            }
        }
        return out.toList()
    }

    private fun fromInterfaces(): List<String> {
        val out = mutableListOf<String>()
        val ifaces = NetworkInterface.getNetworkInterfaces() ?: return out
        for (iface in ifaces) {
            if (!iface.isUp || iface.isLoopback) continue
            val name = iface.name.lowercase()
            if (name.contains("dummy") || name.contains("rmnet") && !name.contains("wlan")) {
                // keep wlan / eth; skip obvious virtual if needed
            }
            for (addr in iface.inetAddresses) {
                if (addr is Inet4Address && !addr.isLoopbackAddress) {
                    out += addr.hostAddress ?: continue
                }
            }
        }
        return out.distinct()
    }
}
