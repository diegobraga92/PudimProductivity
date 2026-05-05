// @generated automatically by Diesel CLI.

diesel::table! {
    lists (id) {
        id -> Uuid,
        parent_id -> Nullable<Uuid>,
        name -> Text,
        list_type -> Int2,
    }
}

diesel::table! {
    task_completions (task_id, date) {
        task_id -> Uuid,
        date -> Timestamptz,
    }
}

diesel::table! {
    tasks (id) {
        id -> Uuid,
        list_id -> Uuid,
        text -> Text,
        done -> Bool,
        repeat_on -> Nullable<Array<Nullable<Int2>>>,
        created_at -> Timestamptz,
        updated_at -> Timestamptz,
        deleted_at -> Nullable<Timestamptz>,
    }
}

diesel::joinable!(task_completions -> tasks (task_id));
diesel::joinable!(tasks -> lists (list_id));

diesel::allow_tables_to_appear_in_same_query!(lists, task_completions, tasks,);
