package com.pudimproductivity.api

import okhttp3.MediaType.Companion.toMediaType
import okhttp3.MultipartBody
import okhttp3.RequestBody.Companion.asRequestBody
import retrofit2.http.Multipart
import retrofit2.http.POST
import retrofit2.http.Part
import java.io.File

/** Result of POST /api/v1/media/scan-isbn (Phase 9b barcode decoding). */
data class ScanResult(val isbn: String)

interface MediaService {
    @Multipart
    @POST("media/scan-isbn")
    suspend fun scanIsbn(@Part image: MultipartBody.Part): ScanResult
}

/** Builds a multipart body part from an image file. */
fun imagePart(file: File, field: String = "image"): MultipartBody.Part {
    val mime = if (file.extension.equals("png", ignoreCase = true)) {
        "image/png"
    } else if (file.extension.equals("gif", ignoreCase = true)) {
        "image/gif"
    } else {
        "image/jpeg"
    }
    val body = file.asRequestBody(mime.toMediaType())
    return MultipartBody.Part.createFormData(field, file.name, body)
}

/** Extension property to access MediaService from ApiClient. */
val ApiClient.mediaService: MediaService
    get() = retrofit.create(MediaService::class.java)
