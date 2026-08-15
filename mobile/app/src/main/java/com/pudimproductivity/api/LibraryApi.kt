package com.pudimproductivity.api

import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path
import retrofit2.http.Query

data class LibraryItem(
    val id: String,
    val name: String,
    val media_type: String,
    val release_year: Int? = null,
    val done: Boolean = false,
    val notes: String = "",
    val subtype: String = "",
    val score: Double? = null,
    val score_source: String = ""
)

data class CreateLibraryItemRequest(
    val name: String,
    val media_type: String,
    val release_year: Int? = null,
    val done: Boolean = false,
    val notes: String = "",
    val subtype: String = "",
    val score: Double? = null,
    val score_source: String = ""
)

data class UpdateLibraryItemRequest(
    val name: String? = null,
    val media_type: String? = null,
    val release_year: Int? = null,
    val done: Boolean? = null,
    val notes: String? = null,
    val subtype: String? = null,
    val score: Double? = null,
    val score_source: String? = null
)

data class ScoreCandidate(
    val title: String,
    val year: Int? = null,
    val score: Double,
    val score_source: String,
    val external_id: String? = null,
    val url: String? = null
)

interface LibraryService {
    @GET("library")
    suspend fun listItems(
        @Query("type") mediaType: String? = null,
        @Query("done") done: Boolean? = null,
        @Query("subtype") subtype: String? = null
    ): List<LibraryItem>

    @GET("library/subtypes")
    suspend fun subtypes(@Query("type") mediaType: String? = null): List<String>

    @POST("library")
    suspend fun createItem(@Body request: CreateLibraryItemRequest): LibraryItem

    @PUT("library/{itemId}")
    suspend fun updateItem(@Path("itemId") itemId: String, @Body request: UpdateLibraryItemRequest): LibraryItem

    @DELETE("library/{itemId}")
    suspend fun deleteItem(@Path("itemId") itemId: String)

    @GET("library/score/search")
    suspend fun searchScores(
        @Query("type") mediaType: String,
        @Query("query") query: String,
        @Query("year") year: Int? = null
    ): List<ScoreCandidate>
}

val ApiClient.libraryService: LibraryService
    get() = retrofit.create(LibraryService::class.java)
