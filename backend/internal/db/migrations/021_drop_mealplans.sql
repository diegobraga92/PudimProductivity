-- Meal planner module removed: drop the Phase 5 tables. DROP IF EXISTS keeps
-- both fresh installs (016 deleted, tables never created) and existing
-- databases (tables created by 016) working. Also removes the retired
-- 'meal_planning' feature flag seeded by migration 002.
DROP TABLE IF EXISTS meal_plan_shopping_list;
DROP TABLE IF EXISTS meal_plan_slots;
DROP TABLE IF EXISTS meal_plans;

DELETE FROM feature_flags WHERE name = 'meal_planning';
