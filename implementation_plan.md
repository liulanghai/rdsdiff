# Implementation Plan - rdsdiff

Goal: Compare two MySQL `CREATE TABLE` statements (Source and Target) and generate the DDL (`ALTER TABLE` statements) explicitly required to transform the Source table structure to match the Target table structure.

## 1. Project Initialization & Dependencies
- [x] Initialize module (`go mod init rdsdiff`) - Done
- [ ] Install dependencies:
    - parser: `github.com/pingcap/tidb/pkg/parser` (Modern TiDB parser location) or `github.com/pingcap/parser`
    - other utils if needed

## 2. Core Structure
- [ ] Create `main.go`:
    - Define CLI flags: `-s <source_sql>`, `-t <target_sql>` (or file paths).
    - Read input SQLs.
    - Call Diff logic.
    - Output resulting DDL.

## 3. Parsing Logic
- [ ] Implement `ParseSQL(sql string)` function:
    - Use `parser.New()`
    - Parse string to `ast.StmtNode`
    - Extract `ast.CreateTableStmt`
    - Return a structured representation (or just the AST node if sufficient) for comparison.

## 4. Diff Logic
- [ ] Create `Diff(source *ast.CreateTableStmt, target *ast.CreateTableStmt)` function.
- [ ] Comparison steps:
    - **Table Name check**: Ensure we are diffing the intended tables (though usually we assume we match them).
    - **Columns**:
        - Identify missing columns (in Target but not Source) -> **ADD**.
        - Identify extra columns (in Source but not Target) -> **DROP**.
        - Identify modified columns (Type, Length, Nullable, Default, Column options) -> **MODIFY/CHANGE**.
    - **Indexes/Keys**:
        - Identify missing indexes (by name or definition) -> **ADD INDEX**.
        - Identify extra indexes -> **DROP INDEX**.
        - Identify modified indexes -> **DROP & ADD**.
    - **Table Options**:
        - Diff Comments, Charset, Collation -> **ALTER TABLE ... options**.

## 5. DDL Generation
- [ ] Implement `GenerateDDL(diffResult)`:
    - Construct valid MySQL `ALTER TABLE` statements.
    - Support combining multiple changes if possible, or list them sequentially.
    - Example: `ALTER TABLE user ADD age int;`

## 6. Testing & Validation
- [ ] Create a test case based on README example.
- [ ] Verify output matches `alter table user add age int;`.
