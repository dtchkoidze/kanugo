# Go Kanban

A Terminal-based Kanban board application built with Go, using the Charm Bubble Tea framework and PostgreSQL for persistent task storage.

## Features

- Interactive terminal UI using [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- Three-column Kanban board (Todo, In Progress, Done)
- Task creation with title and description
- Move tasks between columns
- Delete tasks
- Persistent storage with PostgreSQL
- Keyboard navigation

## Prerequisites

- Go 1.18 or higher
- PostgreSQL database
- Terminal with full color support

## Installation

1. Clone the repository:

```bash
git clone https://github.com/dtchkoidze/kanugo.git
cd go-kanban
```

2. Install dependencies:

```bash
go mod download
```

3. Set up the PostgreSQL database:

First, make sure you have a PostgreSQL database created:

```bash
# Example command to create a database (run in your terminal)
createdb kanban_db
```

Then, run the migration script to create the necessary tables:

```bash
# Run the migration script
go run scripts/migrate.go
```

The migration will create the following table structure:

```
Table "public.tasks"
   Column    |  Type   | Collation | Nullable |              Default              
-------------+---------+-----------+----------+-----------------------------------
 id          | integer |           | not null | nextval('tasks_id_seq'::regclass)
 title       | text    |           | not null | 
 description | text    |           |          | 
 status      | integer |           |          | 
Indexes:
    "tasks_pkey" PRIMARY KEY, btree (id)
```

4. Create a `.env` file in the project root:

```
DATABASE_URL=postgres://username:password@localhost:5432/kanban_db
```

## Usage

Run the application:

```bash
go run main.go
```

### Controls

- `←/→` or `h/l`: Navigate between columns
- `↑/↓`: Navigate tasks within a column
- `Enter`: Move task to the next column (cycles back to Todo after Done)
- `n`: Add a new task
- `Delete`: Delete the selected task
- `q`: Quit the application

## Project Structure

```
go-kanban/
├── main.go                         # Main application code
├── migrations/                     # Database migrations
│   └── 001_create_tasks_table.sql  # Initial table creation
├── scripts/                        # Utility scripts
│   └── migrate.go                  # Database migration helper
├── go.mod                          # Go module definition
├── go.sum                          # Go module checksums
├── .env                            # Environment variables (not in version control)
├── README.md                       # This file
└── .gitignore                      # Git ignore file
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - UI components for Bubble Tea
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Style definitions for terminal applications
- [pgx](https://github.com/jackc/pgx) - PostgreSQL driver
- [godotenv](https://github.com/joho/godotenv) - .env file parser

## Development

To build the application:

```bash
go build -o kanban
```

Or to build && run:
```bash
go run .
```

## License

MIT
