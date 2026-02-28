TODO: Remove this file

# Designing the project and architecture

1. Clarify the problem → shape the domain → define responsibilities → only then pick technology

2. Domain: The subject of the project. What concepts will still exist if the UI, framework, or platform changes?
    e.g.:
    - User
    - Task
    - List
    - Daily List
    - Focus Session (future)
    - Pomodoro Cycle (future)

3. For each concept of the domain, define rules
    Example:
    - A task belongs to exactly one list
    - A daily list is derived from tasks, not manually created
    - A task can be completed only once per day
    - A focus session may or may not be linked to a task

4. Draw Responsabilities:
    - Domain: Enforces Rules
    - Application: Coordinate actions
    - UI: Input and Output
    - Infra: Persists data, and deals with externals

5. Define Structure
    - Define shape of the system
    - Expect how the system will change, and try to contain that
    - Define axes of change. E.g.:
        - UI (Web vs Mobile)
        - Features (Tasks vs Pomodoro vs Analytics)
        - Persistence (DB?)
        - Auth
        - Scale (single vs distributed)
    - Favor simplest structures, but allow extension
    - Helpful questions:
        - Multiple clients? Yes, then API boundary (Backend separate from UI). If no, monolith would be fine
        - Growing feature set? Yes, then use modular codebase. If no, flat structure is fine
        - Large number of users or throughput? Yes, use microservices. If no, monolith is fine
        - Multiple teams? Yes, define boundaries. If no, avoid distributed systems
    - In this case (for now): One deployable unit with internal modules. E.g.: Task module, List module, Focus Module
    - Draw diagram to explain. If it's not possible to explain, might be overly complex.

6. Validation
    - Consider how the project will change, and if that will work.
    - What new concepts will be created? What existing ones will change? What modules will change? Will it extend or change?


# Mermaid Diagram Types and Uses

1. Flowchart (`graph` / `flowchart`)
    - What it’s for
        - High-level architecture
        - Component relationships
        - Data flow
        - Layered structures
    - Typical questions it answers
        - Who talks to whom?
        - Where does data flow?
        - What depends on what?

```mermaid
flowchart LR
  Web --> API
  API --> DB
```


2. Class Diagram (`classDiagram`)
    - What it’s for
        - Domain concepts
        - Relationships
        - Ownership
        - Cardinality (1-to-many, optional, etc.)
    - Typical questions it answers
        - What entities exist?
        - How are they related?
        - Who owns what?

```mermaid
classDiagram
  User "1" --> "many" List
  List "1" --> "many" Task
```

3. Sequence Diagram (`sequenceDiagram`)

    - What it’s for
        - Feature execution
        - Request/response flows
        - Interaction ordering
    - Typical questions it answers
        - What happens when X occurs?
        - In what order do things happen?
        - Who initiates what?

```mermaid
sequenceDiagram
  User->>Web: Click "Complete"
  Web->>API: PATCH /tasks
  API->>Domain: completeTask()
```

4. State Diagram (`stateDiagram-v2`)

    - What it’s for
        - State transitions
        - Allowed vs forbidden actions
        - Business invariants
    - Typical questions it answers
        - What states exist?
        - What transitions are valid?
        - What actions cause transitions?

```mermaid
stateDiagram-v2
  Pending --> Completed : complete()
  Completed --> Pending : reopen()
```