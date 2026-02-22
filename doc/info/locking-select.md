## SELECT ... FOR UPDATE con GORM (locking explícito)

Para hacer un `SELECT ... FOR UPDATE` con GORM en Go, se usa el locking explícito.

En GORM v2 se hace con `clause.Locking`.

### Equivalente en GORM

```go
import (
    "gorm.io/gorm"
    "gorm.io/gorm/clause"
)

var processVersion ProcessVersion

err := db.
    Clauses(clause.Locking{Strength: "UPDATE"}).
    Select("id").
    Where("process_type_id = ? AND status = ?", vProcessTypeID, "PROD").
    First(&processVersion).Error

if err != nil {
    // manejar error
}
```

### SQL generado

Ese código produce algo como:

```sql
SELECT id
FROM process_versions
WHERE process_type_id = ?
  AND status = ?
ORDER BY id
LIMIT 1
FOR UPDATE;
```

`First()` agrega `ORDER BY` y `LIMIT 1`.  
Si querés todos los registros con `FOR UPDATE`, usá `Find()`.

### Obtener todos los IDs bloqueados

```go
var processVersions []ProcessVersion

err := db.
    Clauses(clause.Locking{Strength: "UPDATE"}).
    Select("id").
    Where("process_type_id = ? AND status = ?", vProcessTypeID, "PROD").
    Find(&processVersions).Error
```

### Importante: usar siempre transacción

`FOR UPDATE` solo funciona dentro de una transacción:

```go
err := db.Transaction(func(tx *gorm.DB) error {
    var processVersion ProcessVersion

    if err := tx.
        Clauses(clause.Locking{Strength: "UPDATE"}).
        Select("id").
        Where("process_type_id = ? AND status = ?", vProcessTypeID, "PROD").
        First(&processVersion).Error; err != nil {
        return err
    }

    // lógica adicional aquí...

    return nil
})
```

