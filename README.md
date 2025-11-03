# 🧠 ONQL — Object Notation Query Language
### Context-Aware Database Engine by Autobit Software Services Pvt. Ltd.

> **ONQL (Object Notation Query Language)** is a next-generation, **context-aware database** designed for the modern era of APIs and AI integration.
> It understands relationships **without explicit joins**, making data fetching faster, smarter, and more natural.

---

## 🚀 Overview

ONQL is built to simplify how data is fetched, related, and exposed through APIs.
Traditional databases require **joins, foreign keys, and rigid schema design** to fetch related data.
ONQL introduces **context-awareness**, allowing the engine to automatically understand and retrieve related objects — ideal for **AI-driven systems**, **microservices**, and **API backends**.

---

## ✨ Key Features

- 🧩 **Context-Aware Query Engine** — Fetch related data automatically without manual joins.
- ⚡ **Object Notation Query Language (ONQL)** — A human-friendly query syntax inspired by object notation.
- 🌐 **API-First Design** — Expose data directly to frontends or services without the need for extensive backend code.
- 🤖 **AI & Microservices Ready** — Simplify data consumption for complex, distributed systems.
- 🛡️ **Context-Aware Security** — Define access controls and permissions directly within your data relationships.

---

# ONQL Developer Documentation

**Powering Instant Insights**
*Last updated: October 14, 2025*

## Table of Contents

1.  [Introduction](#1-introduction)
2.  [Installation](#2-installation)
3.  [Schema](#3-schema)
4.  [Object Notation Query Language (ONQL)](#4-object-notation-query-language-onql)
5.  [Protocols](#5-protocols)
6.  [Plugins](#6-plugins)
7.  [Migrations](#7-migrations)
8.  [CLI Client](#8-cli-client)
9.  [Drivers](#9-drivers)
10. [ORMs](#10-orms)
11. [Plugin SDK](#11-plugin-sdk)
12. [Appendices & References](#12-appendices--references)

---

## 1. Introduction

ONQL (Object Notation Query Language) is the world's first context-aware, post-relational database management system. It is engineered to solve common problems in modern software development, such as simplifying API data fetching, streamlining migrations, and enabling seamless integration with AI models.

## 2. Installation

*(This section is a placeholder. Detailed installation instructions will be provided here.)*

## 3. Schema

ONQL schemas are used to define the structure of your databases and tables. They provide the foundational blueprint for your data, upon which you can layer context, relationships, and access controls using [Protocols](#5-protocols).

## 4. Object Notation Query Language (ONQL)

ONQL is a powerful Domain-Specific Language (DSL) that allows you to query your database directly from any client, including frontends. Because ONQL is context-aware, you can fetch complex, related data without writing backend API endpoints.

### Secondary Datatypes

ONQL operates on a set of intuitive, object-based data structures:

| Type        | Description                                      | Example                   |
|:------------|:-------------------------------------------------|:--------------------------|
| **Table**   | An array of objects.                             | `db.accounts`             |
| **Row**     | A single object from a table.                    | `db.accounts[0]`          |
| **List**    | A vertical array of values from a single column. | `db.accounts.username`    |
| **Field**   | A single value from a specific row and column.   | `db.accounts[0].username` |
| **Literal** | A static string or number.                       | `"harry"`, `1234`         |

> **Note:** ONQL does not support boolean literals (`true`/`false`) directly in queries. Instead, use comparison expressions that evaluate to a boolean outcome.

### Operators

ONQL supports a familiar set of operators for complex queries.

| Category       | Operators                        |
|:---------------|:---------------------------------|
| **Arithmetic** | `+`, `-`, `*`, `/`, `%`, `**`    |
| **Comparison** | `<`, `>`, `<=`, `>=`, `!=`, `in` |
| **Logical**    | `and`, `or`, `not`               |
| **Priority**   | `( )`                            |

### Query Examples

```
db.accounts[balance > 0 and name="harry"]
db.transactions[(amount + 100) > 1000]
db.accounts[id in db.accounts[name="harry"].id]
```

### Filtration

Use square brackets `[]` to filter data. Inside the brackets, you can use column names, related table names (defined in a protocol), operators, literals, and even nested queries.

```
// Filter accounts with a total transaction amount over 1000
db.accounts[transactions.amount._sum > 1000]

// Chain filters for more complex queries
db.accounts[name="harry"][age > 30]
```

> **Note:** A nested query must start with the `db` name alias from protocol file; otherwise, ONQL will interpret it as a related table name.

### Relation Access

Use the dot (`.`) operator to access fields, lists, and related tables.

### Slicing

ONQL supports Python-style slicing to retrieve a subset of rows from a table.

```
// Get 10 accounts, starting from index 50
db.accounts[50:60]

// Get every second account from the first 10
db.accounts[0:10:2]
```

Slicing can be applied before or after filtration, as ONQL evaluates queries from left to right.

### Row Access

To access a single row by its index, use a single number in square brackets `[]`.

```
db.accounts[205]
```

Filtration and slicing must be performed *before* row access, as row access converts the `Table` into a `Row`, which cannot be further filtered or sliced.

### Projection

Projections, defined with curly braces `{}`, are the final step in an ONQL query. They allow you to select specific columns and shape the final output.

```
// Select specific fields
db.accounts{name, balance}

// Use aliases for fields
db.accounts{"n": name, "b": balance, age}

// Project data from related tables
db.accounts{name, balance, orders}

// Run nested queries within a projection
db.accounts{name, orders[qty > 20]}
```

### Aggregates

ONQL provides simple, parenthesis-free aggregate functions.

| Function | Description                                | Example                                      |
|----------|--------------------------------------------|----------------------------------------------|
| `_sum`   | Calculates the sum of a list.              | `db.accounts.balance._sum`                   |
| `_max`   | Finds the maximum value in a list.         | `db.transactions.amount._max`                |
| `_min`   | Finds the minimum value in a list.         | `db.transactions.amount._min`                |
| `_asc`   | Sorts a table or list in ascending order.  | `db.accounts._asc(username)`                 |
| `_dsc`   | Sorts a table or list in descending order. | `db.accounts._dsc(created_at)`               |
| `_like`  | Filters a list with a SQL LIKE pattern.    | `db.accounts.name._like("sa%")`              |
| `_date`  | Formats a timestamp.                       | `db.accounts.created_at._date("2006-01-02")` |
| `_count` | Counts the items in a table or list.       | `db.accounts._count`                         |

### Nested Queries

Nested queries can be used within filtrations and projections. To access data from the parent table within a nested query, use the `parent` keyword.

```
db.accounts[id in db.transactions[parent.id == account_id].account_id]
```

## 5. Protocols

Protocols are JSON files that define the relationships between tables and the context for who can access them. While ONQL maintains a default protocol based on your schema, you can create custom protocols to grant specific, aliased, and secure access to your data.

### Key Functions of a Protocol

*   **Relation Mapping:** Declare relationships between tables so the ONQL engine can fetch connected data without manual joins.
*   **Access Context:** Specify who can access what. Define permissions and visibility at the table level.
*   **Developer-Friendly Aliases:** Provide simple aliases for complex table names, enabling clean and readable queries.

### Example Protocol

```json
{
  "db_alias_name": {
    "database": "db_original_name_in_schema",
    "accounts": {
      "table": "accounts",
      "fields": {
        "account_id": "account_id",
        "balance": "balance"
      },
      "relations": {
        "orders": {
          "prototable": "orders",
          "type": "otm",
          "through": "",
          "entity": "orders",
          "fk_field": "id:account_id"
        }
      },
      "context": {
        "account": "fintrabit.accounts[id=$1]"
      }
    }
  }
}
```

### Protocol Relations

ONQL supports four types of relations:

| Type  | Name             | Description                                                                                              |
|:------|:-----------------|:---------------------------------------------------------------------------------------------------------|
| `oto` | **One-to-One**   | Each record in one table is linked to one and only one record in another table.                          |
| `otm` | **One-to-Many**  | A single record in one table can be linked to multiple records in another.                               |
| `mto` | **Many-to-One**  | Multiple records in one table can be linked to a single record in another.                               |
| `mtm` | **Many-to-Many** | Multiple records in one table can be linked to multiple records in another, often via a `through` table. |

**Important Notes:**

*   `entity` and `through` refer to the original table names from the schema.
*   `prototable` is the alias for the table within the current protocol.
*   ONQL queries must use the alias names defined in the protocol, not the original schema names.

## 6. Plugins

Plugins extend ONQL's core functionality, allowing you to introduce custom validation, authorization, and data transformation logic.

## 7. Migrations

ONQL provides a built-in, JSON-based migration system to manage versioned schema changes.

## 8. CLI Client

The ONQL CLI is a powerful command-line interface for direct interaction with your ONQL systems.

## 9. Drivers

Drivers abstract the complexities of communication, connection pooling, and serialization, providing a simple interface for client applications.

## 10. ORMs

Use ONQL as a lightweight Object-Relational Mapper (ORM) or integrate it with your existing ORM tools.

## 11. Plugin SDK

The Plugin SDK provides the tools and interfaces needed to build your own plugins and extend ONQL's behavior.

## 12. Appendices & References

*   **DSL:** JSON-based query notation
*   **Migration:** Versioned schema changes
*   **Plugin:** Extension module
*   **Driver:** Client library
*   **ORM:** Object-Relational Mapping
