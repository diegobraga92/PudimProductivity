package com.pudimproductivity.api

import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path
import retrofit2.http.Query

data class RecipeIngredient(
    val id: String? = null,
    val name: String,
    val quantity: String = "",
    val unit: String = "",
    val sort_order: Int? = null
)

data class RecipeStep(
    val id: String? = null,
    val step_number: Int? = null,
    val instruction: String
)

data class Recipe(
    val id: String,
    val title: String,
    val description: String? = null,
    val difficulty: String = "easy",
    val prep_time_minutes: Int = 0,
    val cook_time_minutes: Int = 0,
    val servings: Int = 1,
    val image_url: String? = null,
    val source_url: String? = null,
    val tags: List<String>? = null,
    val ingredients: List<RecipeIngredient>? = null,
    val steps: List<RecipeStep>? = null
)

data class CreateRecipeRequest(
    val title: String,
    val description: String = "",
    val difficulty: String = "easy",
    val prep_time_minutes: Int = 0,
    val cook_time_minutes: Int = 0,
    val servings: Int = 1,
    val tags: List<String>? = null,
    val ingredients: List<RecipeIngredient> = emptyList(),
    val steps: List<RecipeStep> = emptyList()
)

interface RecipeService {
    @GET("recipes")
    suspend fun listRecipes(
        @Query("search") search: String? = null,
        @Query("tags") tags: String? = null,
        @Query("difficulty") difficulty: String? = null
    ): List<Recipe>

    @GET("recipes/{recipeId}")
    suspend fun getRecipe(@Path("recipeId") recipeId: String): Recipe

    @POST("recipes")
    suspend fun createRecipe(@Body request: CreateRecipeRequest): Recipe

    @DELETE("recipes/{recipeId}")
    suspend fun deleteRecipe(@Path("recipeId") recipeId: String)
}

val ApiClient.recipeService: RecipeService
    get() = retrofit.create(RecipeService::class.java)
