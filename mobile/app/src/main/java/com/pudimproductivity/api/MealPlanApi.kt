package com.pudimproductivity.api

import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path

data class MealSlot(
    val id: String? = null,
    val date: String,
    val meal_type: String,
    val recipe_id: String? = null,
    val notes: String? = null
)

data class MealPlan(
    val id: String,
    val name: String? = null,
    val start_date: String,
    val end_date: String,
    val is_published: Boolean = false,
    val slots: List<MealSlot>? = null
)

data class CreateMealPlanRequest(
    val name: String = "",
    val start_date: String,
    val end_date: String,
    val slots: List<MealSlot> = emptyList()
)

data class ShoppingItem(
    val id: String,
    val ingredient_name: String,
    val quantity_agg: String = "",
    val unit: String = "",
    val is_checked: Boolean = false
)

interface MealPlanService {
    @GET("mealplans")
    suspend fun listMealPlans(): List<MealPlan>

    @POST("mealplans")
    suspend fun createMealPlan(@Body request: CreateMealPlanRequest): MealPlan

    @GET("mealplans/{planId}")
    suspend fun getMealPlan(@Path("planId") planId: String): MealPlan

    @DELETE("mealplans/{planId}")
    suspend fun deleteMealPlan(@Path("planId") planId: String)

    @POST("mealplans/{planId}/shopping-list")
    suspend fun generateShoppingList(@Path("planId") planId: String): List<ShoppingItem>

    @GET("mealplans/{planId}/shopping-list")
    suspend fun getShoppingList(@Path("planId") planId: String): List<ShoppingItem>
}

val ApiClient.mealPlanService: MealPlanService
    get() = retrofit.create(MealPlanService::class.java)
