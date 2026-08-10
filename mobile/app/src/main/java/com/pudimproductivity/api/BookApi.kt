package com.pudimproductivity.api

import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path
import retrofit2.http.Query

data class Book(
    val id: String,
    val isbn: String,
    val title: String,
    val authors: List<String> = emptyList(),
    val publisher: String? = null,
    val published_date: String? = null,
    val description: String? = null,
    val page_count: Int = 0,
    val thumbnail_url: String? = null,
    val status: String = "want_to_read"
)

data class AddByISBNRequest(val isbn: String)

data class AddManualBookRequest(
    val isbn: String,
    val title: String,
    val authors: List<String> = emptyList()
)

interface BookService {
    @GET("books")
    suspend fun listBooks(@Query("status") status: String? = null): List<Book>

    @POST("books/by-isbn")
    suspend fun addByISBN(@Body request: AddByISBNRequest): Book

    @POST("books")
    suspend fun addManual(@Body request: AddManualBookRequest): Book

    @PUT("books/{bookId}/status")
    suspend fun updateStatus(@Path("bookId") bookId: String, @Body request: UpdateStatusRequest)

    @DELETE("books/{bookId}")
    suspend fun deleteBook(@Path("bookId") bookId: String)
}

data class UpdateStatusRequest(val status: String)

val ApiClient.bookService: BookService
    get() = retrofit.create(BookService::class.java)
