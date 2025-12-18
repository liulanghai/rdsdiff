# rdsdiff
对比MySQL的2个 CREATE TABLE 语句, 生成一个DDL, DDL执行以后则2个CREATE TABLE 语句则完全一致。

## example
```sql
create table user (
    name varchar(100)
   
) comment='user table';

```
```sql 
create table user (
    name varchar(200)
    age int 
) comment='user table';

```

则生成的DDL为 alter table user add age int;


## Usage

### Build
```bash
go build -o rdsdiff
```

### Run
```bash
./rdsdiff -s "create table user (name varchar(100))" -d "create table user (name varchar(200), age int)"
```

### CLI Options
- `-s`: Source CREATE TABLE statement.
- `-d`: Target/Destination CREATE TABLE statement.

## Implementation Details
1. Uses `github.com/pingcap/tidb/pkg/parser` to parse SQL into AST.
2. Compares AST nodes for Columns and Indexes/Constraints.
3. Generates DDL using `ALTER TABLE`.

## Example
Source:
```sql
create table user (
    name varchar(100)
) comment='user table';
```

Target:
```sql
create table user (
    name varchar(200),
    age int 
) comment='user table';
```

Output:
```sql
ALTER TABLE user MODIFY COLUMN `name` VARCHAR(200);
ALTER TABLE user ADD COLUMN `age` INT;
```

