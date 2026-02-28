package com.example.opentodo

import android.os.Bundle
import android.view.View
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import androidx.viewpager2.widget.ViewPager2
import com.example.opentodo.ui.SessionPagerAdapter
import com.google.android.material.tabs.TabLayout
import com.google.android.material.tabs.TabLayoutMediator
import java.net.HttpURLConnection
import java.net.URL
import java.util.concurrent.Executors
import java.util.concurrent.ScheduledExecutorService
import java.util.concurrent.TimeUnit

class MainActivity : AppCompatActivity() {
    companion object {
        private const val CONNECTION_TIMEOUT_MS = 1500
        private const val READ_TIMEOUT_MS = 1500
        private const val BACKEND_CHECK_INTERVAL_SECONDS = 5L
    }

    private lateinit var backendWarningBanner: TextView
    private var backendMonitor: ScheduledExecutorService? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        val viewPager = findViewById<ViewPager2>(R.id.viewPager)
        val tabLayout = findViewById<TabLayout>(R.id.tabLayout)
        backendWarningBanner = findViewById(R.id.backendWarningBanner)

        val pageTitles = listOf("Daily", "Todo", "List of Lists")
        viewPager.adapter = SessionPagerAdapter(this)

        TabLayoutMediator(tabLayout, viewPager) { tab, position ->
            tab.text = pageTitles[position]
        }.attach()

        startBackendMonitoring()
    }

    override fun onDestroy() {
        super.onDestroy()
        backendMonitor?.shutdownNow()
        backendMonitor = null
    }

    private fun startBackendMonitoring() {
        backendMonitor?.shutdownNow()
        backendMonitor = Executors.newSingleThreadScheduledExecutor()
        backendMonitor?.scheduleAtFixedRate(
            {
                val backendAvailable = isBackendAvailable()
                runOnUiThread {
                    backendWarningBanner.visibility =
                        if (backendAvailable) View.GONE else View.VISIBLE
                }
            },
            0,
            BACKEND_CHECK_INTERVAL_SECONDS,
            TimeUnit.SECONDS,
        )
    }

    private fun isBackendAvailable(): Boolean {
        return try {
            val connection = (URL(BuildConfig.BACKEND_URL).openConnection() as HttpURLConnection).apply {
                requestMethod = "GET"
                connectTimeout = CONNECTION_TIMEOUT_MS
                readTimeout = READ_TIMEOUT_MS
            }

            connection.connect()
            connection.responseCode in 200..499
        } catch (_: Exception) {
            false
        }
    }
}
// TODO
