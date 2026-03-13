use crate::domain::list::{List as DomainList, ListType};
use crate::infrastructure::schema::lists;
use diesel::{
    prelude::*,
    r2d2::{ConnectionManager, Pool},
    PgConnection,
};
use uuid::Uuid;

pub type DbPool = Pool<ConnectionManager<PgConnection>>;

// Database model for lists
#[derive(Queryable, Selectable)]
#[diesel(table_name = lists)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct ListModel {
    pub id: Uuid,
    pub parent_id: Option<Uuid>,
    pub name: String,
    pub list_type: i16,
}

// Insertable list
#[derive(Insertable)]
#[diesel(table_name = lists)]
pub struct NewListModel<'a> {
    pub id: &'a Uuid,
    pub parent_id: Option<&'a Uuid>,
    pub name: &'a str,
    pub list_type: i16,
}

#[derive(Clone)]
pub struct ListRepository {
    pool: DbPool,
}

impl ListRepository {
    pub fn new(pool: DbPool) -> Self {
        Self { pool }
    }

    pub fn get_all(&self) -> Result<Vec<DomainList>, diesel::result::Error> {
        let mut conn = self.pool.get().unwrap();
        let results: Vec<ListModel> = lists::table.order_by(lists::name.asc()).load(&mut conn)?;

        Ok(results.into_iter().map(Self::to_domain).collect())
    }

    pub fn create(&self, list: DomainList) -> Result<DomainList, diesel::result::Error> {
        let mut conn = self.pool.get().unwrap();

        let new_list = NewListModel {
            id: &list.id,
            parent_id: list.parent_id.as_ref(),
            name: &list.name,
            list_type: Self::list_type_to_i16(&list.list_type),
        };

        diesel::insert_into(lists::table)
            .values(&new_list)
            .execute(&mut conn)?;

        Ok(list)
    }

    pub fn find_by_id(&self, id: Uuid) -> Result<Option<DomainList>, diesel::result::Error> {
        let mut conn = self.pool.get().unwrap();
        let result: Option<ListModel> = lists::table
            .filter(lists::id.eq(id))
            .first(&mut conn)
            .optional()?;

        Ok(result.map(Self::to_domain))
    }

    fn to_domain(model: ListModel) -> DomainList {
        DomainList {
            id: model.id,
            parent_id: model.parent_id,
            name: model.name,
            list_type: Self::i16_to_list_type(model.list_type),
        }
    }

    fn list_type_to_i16(list_type: &ListType) -> i16 {
        match list_type {
            ListType::Todo => 0,
            ListType::Daily => 1,
            ListType::Collection => 2,
        }
    }

    fn i16_to_list_type(value: i16) -> ListType {
        match value {
            0 => ListType::Todo,
            1 => ListType::Daily,
            2 => ListType::Collection,
            _ => ListType::Todo, // default
        }
    }
}

// TODO REVIEW
