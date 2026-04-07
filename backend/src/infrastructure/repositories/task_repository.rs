use crate::domain::task::{Task, TaskCompletion};
use crate::infrastructure::schema::{task_completions, tasks};
use chrono::{DateTime, Utc};
use diesel::{
    prelude::*,
    r2d2::{ConnectionManager, Pool},
    PgConnection,
};
use uuid::Uuid;

pub type DbPool = Pool<ConnectionManager<PgConnection>>;

// Database model for tasks
#[derive(Queryable, Selectable)]
#[diesel(table_name = tasks)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct TaskModel {
    pub id: Uuid,
    pub list_id: Uuid,
    pub text: String,
    pub done: bool,
    pub repeat_on: Option<Vec<Option<i16>>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub deleted_at: Option<DateTime<Utc>>,
}

// Insertable task
#[derive(Insertable)]
#[diesel(table_name = tasks)]
pub struct NewTaskModel<'a> {
    pub id: &'a Uuid,
    pub list_id: &'a Uuid,
    pub text: &'a str,
    pub done: bool,
    pub repeat_on: Option<&'a Vec<Option<i16>>>,
    pub created_at: &'a DateTime<Utc>,
    pub updated_at: &'a DateTime<Utc>,
    pub deleted_at: Option<&'a DateTime<Utc>>,
}

// Database model for task completions
#[derive(Queryable, Selectable)]
#[diesel(table_name = task_completions)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct TaskCompletionModel {
    pub task_id: Uuid,
    pub date: DateTime<Utc>,
}

// Insertable task completion
#[derive(Insertable)]
#[diesel(table_name = task_completions)]
pub struct NewTaskCompletionModel<'a> {
    pub task_id: &'a Uuid,
    pub date: &'a DateTime<Utc>,
}

#[derive(Clone)]
pub struct TaskRepository {
    pool: DbPool,
}

impl TaskRepository {
    pub fn new(pool: DbPool) -> Self {
        Self { pool }
    }

    pub fn get_by_list_id(&self, list_id: Uuid) -> Result<Vec<Task>, diesel::result::Error> {
        let mut conn = self.pool.get().unwrap();
        let results: Vec<TaskModel> = tasks::table
            .filter(tasks::list_id.eq(list_id))
            .filter(tasks::deleted_at.is_null())
            .order_by(tasks::created_at.asc())
            .load(&mut conn)?;

        // Get task completions for these tasks
        let task_ids: Vec<Uuid> = results.iter().map(|task| task.id).collect();
        let completions = self.get_completions_by_task_ids(&task_ids)?;

        Ok(results
            .into_iter()
            .map(|model| Self::to_domain(model, &completions))
            .collect())
    }

    pub fn create(&self, task: Task) -> Result<Task, diesel::result::Error> {
        let mut conn = self.pool.get().unwrap();

        let repeat_on_vec = task.repeat_on.as_ref().map(|weekdays| {
            weekdays
                .iter()
                .map(|wd| Some(*wd as i16))
                .collect::<Vec<Option<i16>>>()
        });

        let new_task = NewTaskModel {
            id: &task.id,
            list_id: &task.list_id,
            text: &task.text,
            done: task.done,
            repeat_on: repeat_on_vec.as_ref(),
            created_at: &task.created_at.unwrap_or_else(Utc::now),
            updated_at: &Utc::now(),
            deleted_at: task.deleted_at.as_ref(),
        };

        diesel::insert_into(tasks::table)
            .values(&new_task)
            .execute(&mut conn)?;

        // If task is completed, create a completion record
        if task.done {
            let completion = TaskCompletion {
                task_id: task.id,
                date: task.created_at.unwrap_or_else(Utc::now),
            };
            self.create_completion(&completion)?;
        }

        Ok(task)
    }

    pub fn find_by_id(&self, id: Uuid) -> Result<Option<Task>, diesel::result::Error> {
        let mut conn = self.pool.get().unwrap();
        let result: Option<TaskModel> = tasks::table
            .filter(tasks::id.eq(id))
            .filter(tasks::deleted_at.is_null())
            .first(&mut conn)
            .optional()?;

        if let Some(model) = result {
            let completions = self.get_completions_by_task_ids(&[id])?;
            Ok(Some(Self::to_domain(model, &completions)))
        } else {
            Ok(None)
        }
    }

    pub fn update_completion(
        &self,
        task_id: Uuid,
        completed: bool,
    ) -> Result<Task, diesel::result::Error> {
        let mut conn = self.pool.get().unwrap();
        let updated_at = Utc::now();

        // Update the task's done status
        let num_updated = diesel::update(tasks::table.filter(tasks::id.eq(task_id)))
            .set((tasks::done.eq(completed), tasks::updated_at.eq(updated_at)))
            .execute(&mut conn)?;

        if num_updated == 0 {
            return Err(diesel::result::Error::NotFound);
        }

        // Handle task completions
        if completed {
            let completion = TaskCompletion {
                task_id,
                date: updated_at,
            };
            self.create_completion(&completion)?;
        } else {
            // Remove completion records
            diesel::delete(task_completions::table.filter(task_completions::task_id.eq(task_id)))
                .execute(&mut conn)?;
        }

        // Get the updated task
        self.find_by_id(task_id)
            .and_then(|opt| opt.ok_or(diesel::result::Error::NotFound))
    }

    pub fn soft_delete(&self, task_id: Uuid) -> Result<(), diesel::result::Error> {
        let mut conn = self.pool.get().unwrap();
        let updated_at = Utc::now();

        diesel::update(tasks::table.filter(tasks::id.eq(task_id)))
            .set((
                tasks::deleted_at.eq(Some(updated_at)),
                tasks::updated_at.eq(updated_at),
            ))
            .execute(&mut conn)?;

        Ok(())
    }

    fn create_completion(&self, completion: &TaskCompletion) -> Result<(), diesel::result::Error> {
        let mut conn = self.pool.get().unwrap();

        let new_completion = NewTaskCompletionModel {
            task_id: &completion.task_id,
            date: &completion.date,
        };

        diesel::insert_into(task_completions::table)
            .values(&new_completion)
            .execute(&mut conn)?;

        Ok(())
    }

    fn get_completions_by_task_ids(
        &self,
        task_ids: &[Uuid],
    ) -> Result<Vec<TaskCompletion>, diesel::result::Error> {
        if task_ids.is_empty() {
            return Ok(Vec::new());
        }

        let mut conn = self.pool.get().unwrap();
        let results: Vec<TaskCompletionModel> = task_completions::table
            .filter(task_completions::task_id.eq_any(task_ids))
            .order_by(task_completions::date.desc())
            .load(&mut conn)?;

        Ok(results
            .into_iter()
            .map(Self::completion_to_domain)
            .collect())
    }

    fn to_domain(model: TaskModel, completions: &[TaskCompletion]) -> Task {
        // Find most recent completion for this task
        let completion = completions
            .iter()
            .find(|c| c.task_id == model.id)
            .map(|c| c.date);

        Task {
            id: model.id,
            list_id: model.list_id,
            text: model.text,
            done: model.done,
            repeat_on: model.repeat_on.map(|repeat_on| {
                repeat_on
                    .into_iter()
                    .filter_map(|opt| opt.map(|n| chrono::Weekday::try_from(n as u8).ok()))
                    .flatten()
                    .collect()
            }),
            created_at: Some(model.created_at),
            deleted_at: model.deleted_at,
        }
    }

    fn completion_to_domain(model: TaskCompletionModel) -> TaskCompletion {
        TaskCompletion {
            task_id: model.task_id,
            date: model.date,
        }
    }
}
