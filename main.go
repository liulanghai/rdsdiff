package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"
	_ "github.com/pingcap/tidb/pkg/parser/test_driver"

)

func main() {
	srcSQL := flag.String("s", "", "Source CREATE TABLE SQL")
	dstSQL := flag.String("d", "", "Dest/Target CREATE TABLE SQL (the desired state)")
	// Or file paths
	// For simplicity, let's treat them as raw strings for now or add file support if needed.
	// But usually passing large SQL strings in flags is annoying. 
	// Let's also support simply passing hardcoded strings in code for testing 
	// or reading from file if strings start with @. 
	// For now, let's assume valid SQL text is passed.
	flag.Parse()

	if *srcSQL == "" || *dstSQL == "" {
		// Use the example from README if no args provided, for testing purposes
		fmt.Println("No args provided, using default example...")
		*srcSQL = `create table user (
    name varchar(100)
) comment='user table';`
		*dstSQL = `create table user (
    name varchar(200),
    age int 
) comment='user table';`
	}

	fmt.Printf("Source: %s\n", *srcSQL)
	fmt.Printf("Target: %s\n", *dstSQL)

	srcNode, err := parseOne(*srcSQL)
	if err != nil {
		log.Fatalf("Failed to parse source SQL: %v", err)
	}
	dstNode, err := parseOne(*dstSQL)
	if err != nil {
		log.Fatalf("Failed to parse target SQL: %v", err)
	}

	diffs := diffNodes(srcNode, dstNode)
	if len(diffs) > 0 {
		fmt.Println("\nGenerated DDL:")
		for _, ddl := range diffs {
			fmt.Println(ddl)
		}
	} else {
		fmt.Println("\nNo changes needed.")
	}
}

func parseOne(sql string) (*ast.CreateTableStmt, error) {
	p := parser.New()
	stmtNodes, _, err := p.Parse(sql, "", "")
	if err != nil {
		return nil, err
	}
	if len(stmtNodes) != 1 {
		return nil, fmt.Errorf("expected exactly one statement, got %d", len(stmtNodes))
	}
	createStmt, ok := stmtNodes[0].(*ast.CreateTableStmt)
	if !ok {
		return nil, fmt.Errorf("statement is not a CREATE TABLE statement")
	}
	return createStmt, nil
}

func diffNodes(src, dst *ast.CreateTableStmt) []string {
	var ddls []string
	tableName := src.Table.Name.O

	// 1. Check Columns
	// Map src/dst columns for easy lookup
	srcCols := make(map[string]*ast.ColumnDef)
	for _, col := range src.Cols {
		srcCols[col.Name.Name.O] = col
	}
	dstCols := make(map[string]*ast.ColumnDef)
	for _, col := range dst.Cols {
		dstCols[col.Name.Name.O] = col
	}

	// Check for ADD (in dst but not src) or MODIFY (in both)
	for name, dstCol := range dstCols {
		srcCol, exists := srcCols[name]
		if !exists {
			// ADD COLUMN
			colDef := restore(dstCol)
			ddls = append(ddls, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;", tableName, colDef))
		} else {
			// Compare using Restore
			srcDef := restore(srcCol)
			dstDef := restore(dstCol)
			if srcDef != dstDef {
				// Modified
				// Note: MODIFY COLUMN needs the definition but we should be careful.
				// Restore returns "colName type options", which is what MODIFY expects mostly (except primary key constraints sometimes being separate).
				// We might need to trip the column name if Restore includes it? 
				// ast.ColumnDef Restore usually includes the name.
				ddls = append(ddls, fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s;", tableName, dstDef))
			}
		}
	}

	// Check for DROP (in src but not dst)
	for name := range srcCols {
		if _, exists := dstCols[name]; !exists {
			ddls = append(ddls, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", tableName, name))
		}
	}

	// 2. Check Constraints / Indexes
	srcConstraints := make(map[string]*ast.Constraint)
	for _, c := range src.Constraints {
		name := c.Name
		if c.Tp == ast.ConstraintPrimaryKey {
			name = "__PK__"
		} else if name == "" {
			// Fallback: use columns to generate a pseudo-key?
			// For now, let's assume valid named indexes or just use restore signature as ID?
			// But we need to match src to dst.
			// Let's Skip un-named indexes to avoid issues for now in this v1.
			// Or log a warning.
			continue 
		}
		srcConstraints[name] = c
	}
	dstConstraints := make(map[string]*ast.Constraint)
	for _, c := range dst.Constraints {
		name := c.Name
		if c.Tp == ast.ConstraintPrimaryKey {
			name = "__PK__"
		} else if name == "" {
			continue
		}
		dstConstraints[name] = c
	}

	for name, dstC := range dstConstraints {
		srcC, exists := srcConstraints[name]
		if !exists {
			ddls = append(ddls, fmt.Sprintf("ALTER TABLE %s ADD %s;", tableName, restore(dstC)))
		} else {
			if restore(srcC) != restore(dstC) {
				if srcC.Tp == ast.ConstraintPrimaryKey {
					ddls = append(ddls, fmt.Sprintf("ALTER TABLE %s DROP PRIMARY KEY;", tableName))
				} else {
					ddls = append(ddls, fmt.Sprintf("ALTER TABLE %s DROP INDEX %s;", tableName, name))
				}
				ddls = append(ddls, fmt.Sprintf("ALTER TABLE %s ADD %s;", tableName, restore(dstC)))
			}
		}
	}
	
	for name, srcC := range srcConstraints {
		if _, exists := dstConstraints[name]; !exists {
			if srcC.Tp == ast.ConstraintPrimaryKey {
				ddls = append(ddls, fmt.Sprintf("ALTER TABLE %s DROP PRIMARY KEY;", tableName))
			} else {
				ddls = append(ddls, fmt.Sprintf("ALTER TABLE %s DROP INDEX %s;", tableName, name))
			}
		}
	}
	
	return ddls
}

func restore(node ast.Node) string {
	var sb strings.Builder
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)
	if err := node.Restore(ctx); err != nil {
		return ""
	}
	return sb.String()
}

