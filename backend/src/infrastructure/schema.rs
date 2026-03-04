// @generated automatically by Diesel CLI.

diesel::table! {
    lists (id) {
        id -> Text,
        user_id -> Text,
        name -> Text,
        #[sql_name = "type"]
        type_ -> Text,
        config -> Text,
        created_at -> Timestamptz,
        updated_at -> Timestamptz,
    }
}

diesel::table! {
    tasks (id) {
        id -> Text,
        list_id -> Text,
        title -> Text,
        completed -> Bool,
        order_index -> Int4,
        due_date -> Nullable<Timestamptz>,
        recurrence -> Nullable<Text>,
        streak_count -> Nullable<Int4>,
        completed_at -> Nullable<Timestamptz>,
        created_at -> Timestamptz,
        updated_at -> Timestamptz,
    }
}

diesel::joinable!(tasks -> lists (list_id));

diesel::allow_tables_to_appear_in_same_query!(lists, tasks,);
//TODO
