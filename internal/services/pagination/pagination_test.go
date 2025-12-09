package pagination

import (
	"database/sql"
	"errors"
	"go-fiber-core/internal/dtos"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// --- Modelos de Prueba ---
type Profile struct {
	ID     uint
	UserID uint
	Status string
}

type Bank struct {
	ID         uint
	Name       string
	EntityCode string `gorm:"column:entity_code"`
}

type Offer struct {
	ID        uint
	DateOffer time.Time `gorm:"column:date_offer"`
	CodeOffer string    `gorm:"column:code_offer"`
}

type Product struct {
	ID       uint
	Name     string
	Amount   float64
	Cantidad int
	OfferID  uint
	Offer    Offer
	Banks    []*Bank `gorm:"many2many:sales;"`
}

type Sale struct {
	ID          uint
	ProductID   uint      `gorm:"primaryKey"`
	BankID      uint      `gorm:"primaryKey"`
	PaymentDate time.Time `gorm:"column:payment_date"`
}

func (Sale) TableName() string {
	return "sales"
}

type User struct {
	ID        uint
	Name      string `gorm:"column:user_name"`
	Email     string
	IsActive  bool
	CreatedAt time.Time `filter:"date"`
	Profile   Profile
}

// --- Helper para configurar la DB de prueba ---
func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	require.NoError(t, err)

	return gormDB, mock
}

// --- Tests ---

func TestNewPaginationService(t *testing.T) {
	service := NewPaginationService[User]()
	assert.NotNil(t, service, "El servicio de paginación no debería ser nulo")
}

func TestGetDateColumnsFromModel(t *testing.T) {
	service := NewPaginationService[User]()
	dateColumns := service.GetDateColumnsFromModel(&User{})
	expected := map[string]bool{"created_at": true}
	assert.Equal(t, expected, dateColumns, "Debería identificar correctamente las columnas de fecha a través de tags")
}

func TestApplyFilters(t *testing.T) {
	service := NewPaginationService[User]()
	dateColumns := service.GetDateColumnsFromModel(&User{})

	t.Run("Filtro simple de igualdad (case-insensitive)", func(t *testing.T) {
		db, mock := setupTestDB(t)
		req := dtos.PaginationRequest{
			FilterBy:     []string{"user_name"},
			FilterValues: []any{"test"},
		}
		mock.ExpectQuery(`SELECT .* FROM "users" WHERE LOWER\(user_name\) = LOWER\(\$1\)`).
			WithArgs("test").
			WillReturnRows(sqlmock.NewRows([]string{"id"}))
		service.ApplyFilters(db.Model(&User{}), req, dateColumns).Find(&[]User{})
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Filtro por rango de fechas", func(t *testing.T) {
		db, mock := setupTestDB(t)
		req := dtos.PaginationRequest{
			FilterBy:     []string{"created_at"},
			FilterValues: []any{[]any{"2025-01-01", "2025-01-31"}},
		}
		start, _ := time.Parse("2006-01-02", "2025-01-01")
		end, _ := time.Parse("2006-01-02", "2025-01-31")
		endOfDay := end.Add(24 * time.Hour)
		mock.ExpectQuery(`SELECT .* FROM "users" WHERE created_at >= \$1 AND created_at < \$2`).
			WithArgs(start, endOfDay).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))
		service.ApplyFilters(db.Model(&User{}), req, dateColumns).Find(&[]User{})
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Filtro fuzzy en Postgres", func(t *testing.T) {
		db, mock := setupTestDB(t)
		req := dtos.PaginationRequest{
			FilterBy:     []string{"user_name:fuzzy"},
			FilterValues: []any{"test"},
		}
		mock.ExpectQuery(`SELECT .* FROM "users" WHERE user_name % \$1`).
			WithArgs("test").
			WillReturnRows(sqlmock.NewRows([]string{"id"}))
		service.ApplyFilters(db.Model(&User{}), req, dateColumns).Find(&[]User{})
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Filtro LIKE (con ILIKE para Postgres)", func(t *testing.T) {
		db, mock := setupTestDB(t)
		req := dtos.PaginationRequest{
			FilterBy:     []string{"email"},
			FilterValues: []any{"%test.com%"},
		}
		mock.ExpectQuery(`SELECT .* FROM "users" WHERE email ILIKE \$1`).
			WithArgs("%test.com%").
			WillReturnRows(sqlmock.NewRows([]string{"id"}))
		service.ApplyFilters(db.Model(&User{}), req, dateColumns).Find(&[]User{})
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Filtro con cláusula IN", func(t *testing.T) {
		db, mock := setupTestDB(t)
		req := dtos.PaginationRequest{
			FilterBy:     []string{"id"},
			FilterValues: []any{[]any{1, 2, 3}},
		}
		mock.ExpectQuery(`SELECT .* FROM "users" WHERE id IN \(\$1,\$2,\$3\)`).
			WithArgs(1, 2, 3).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))
		service.ApplyFilters(db.Model(&User{}), req, dateColumns).Find(&[]User{})
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Filtro con join a una relación", func(t *testing.T) {
		db, mock := setupTestDB(t)
		req := dtos.PaginationRequest{
			FilterBy:     []string{"profile.status"},
			FilterValues: []any{"active"},
		}
		mock.ExpectQuery(`SELECT .* FROM "users" LEFT JOIN "profiles" "Profile" ON "users"."id" = "Profile"."user_id" WHERE profiles.status = \$1`).
			WithArgs("active").
			WillReturnRows(sqlmock.NewRows([]string{"id"}))
		service.ApplyFilters(db.Model(&User{}), req, dateColumns).Find(&[]User{})
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestApplyOrder(t *testing.T) {
	service := NewPaginationService[User]()

	t.Run("Ordenamiento simple ASC", func(t *testing.T) {
		db, mock := setupTestDB(t)
		req := dtos.PaginationRequest{
			SortBy:   []string{"user_name"},
			SortDesc: []bool{false},
		}
		mock.ExpectQuery(`SELECT .* FROM "users" ORDER BY user_name ASC`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))
		service.ApplyOrder(db.Model(&User{}), req).Find(&[]User{})
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Ordenamiento simple DESC", func(t *testing.T) {
		db, mock := setupTestDB(t)
		req := dtos.PaginationRequest{
			SortBy:   []string{"created_at"},
			SortDesc: []bool{true},
		}
		mock.ExpectQuery(`SELECT .* FROM "users" ORDER BY created_at DESC`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))
		service.ApplyOrder(db.Model(&User{}), req).Find(&[]User{})
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Ordenamiento con join", func(t *testing.T) {
		db, mock := setupTestDB(t)
		req := dtos.PaginationRequest{
			SortBy:   []string{"profile.status"},
			SortDesc: []bool{false},
		}
		mock.ExpectQuery(`SELECT .* FROM "users" LEFT JOIN "profiles" "Profile" ON "users"."id" = "Profile"."user_id" ORDER BY profiles.status ASC`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))
		service.ApplyOrder(db.Model(&User{}), req).Find(&[]User{})
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestExecute(t *testing.T) {
	t.Run("Paginación estándar exitosa", func(t *testing.T) {
		service := NewPaginationService[User]()
		db, mock := setupTestDB(t)
		req := dtos.PaginationRequest{Page: 2, RowsPerPage: 5}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(22))

		rows := sqlmock.NewRows([]string{"id", "user_name"}).AddRow(6, "User 6").AddRow(7, "User 7")
		mock.ExpectQuery(`SELECT \* FROM "users" ORDER BY "users"\."id" ASC LIMIT \$1 OFFSET \$2`).
			WithArgs(5, 5).
			WillReturnRows(rows)

		resp, err := service.Execute(db, req, nil, nil)

		require.NoError(t, err)
		assert.Equal(t, int64(22), resp.TotalRows)
		assert.Len(t, resp.Data, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Sin resultados", func(t *testing.T) {
		service := NewPaginationService[User]()
		db, mock := setupTestDB(t)
		req := dtos.PaginationRequest{Page: 1, RowsPerPage: 10}
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		resp, err := service.Execute(db, req, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(0), resp.TotalRows)
		assert.Len(t, resp.Data, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Error en la consulta de conteo", func(t *testing.T) {
		service := NewPaginationService[User]()
		db, mock := setupTestDB(t)
		req := dtos.PaginationRequest{Page: 1, RowsPerPage: 10}
		dbError := errors.New("db count error")
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).WillReturnError(dbError)
		_, err := service.Execute(db, req, nil, nil)
		require.Error(t, err)
		assert.Equal(t, dbError, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Execute con extras calculator", func(t *testing.T) {
		service := NewPaginationService[User]()
		db, mock := setupTestDB(t)
		req := dtos.PaginationRequest{Page: 1, RowsPerPage: 10}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT SUM(id) as total_sum FROM "users"`)).WillReturnRows(sqlmock.NewRows([]string{"total_sum"}).AddRow(55))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
		mock.ExpectQuery(`SELECT \* FROM "users" ORDER BY "users"\."id" ASC LIMIT \$1`).
			WithArgs(10).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		extrasCalc := func(query *gorm.DB) (map[string]any, error) {
			var result struct{ TotalSum sql.NullInt64 }
			err := query.Select("SUM(id) as total_sum").Scan(&result).Error
			if err != nil {
				return nil, err
			}
			return map[string]any{"total_sum": result.TotalSum.Int64}, nil
		}

		resp, err := service.Execute(db, req, nil, extrasCalc)
		require.NoError(t, err)
		assert.NotNil(t, resp.Extras)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Filtro y orden con relaciones anidadas (usando modifier)", func(t *testing.T) {
		service := NewPaginationService[Product]()
		db, mock := setupTestDB(t)

		req := dtos.PaginationRequest{
			Page:         1,
			RowsPerPage:  10,
			FilterBy:     []string{"offers::code_offer", "banks::entity_code"},
			FilterValues: []any{"OFERTA2025", "B001"},
			SortBy:       []string{"banks::entity_code"},
			SortDesc:     []bool{true},
		}

		modifier := func(query *gorm.DB) *gorm.DB {
			return query.
				Joins("JOIN sales on sales.product_id = products.id").
				Joins("JOIN banks on banks.id = sales.bank_id").
				Joins("JOIN offers on offers.id = products.offer_id")
		}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "products" JOIN sales on sales.product_id = products.id JOIN banks on banks.id = sales.bank_id JOIN offers on offers.id = products.offer_id WHERE offers.code_offer = $1 AND banks.entity_code = $2`)).
			WithArgs("OFERTA2025", "B001").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		rows := sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Producto Test")
		mock.ExpectQuery(`SELECT .* FROM "products" JOIN sales on sales.product_id = products.id JOIN banks on banks.id = sales.bank_id JOIN offers on offers.id = products.offer_id WHERE offers.code_offer = \$1 AND banks.entity_code = \$2 ORDER BY banks.entity_code DESC LIMIT \$3`).
			WithArgs("OFERTA2025", "B001", 10).
			WillReturnRows(rows)

		_, err := service.Execute(db, req, modifier, nil)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet(), "Las expectativas de SQL para joins anidados no se cumplieron")
	})

	t.Run("Deferred Join NO se activa en página 10", func(t *testing.T) {
		service := NewPaginationService[User]()
		db, mock := setupTestDB(t)
		req := dtos.PaginationRequest{Page: 10, RowsPerPage: 10, OptimizeWithKey: "id"}
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(150))
		mock.ExpectQuery(`SELECT \* FROM "users" ORDER BY "users"\."id" ASC LIMIT \$1 OFFSET \$2`).
			WithArgs(10, 90).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(91))
		_, err := service.Execute(db, req, nil, nil)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Deferred Join SÍ se activa en página 11", func(t *testing.T) {
		service := NewPaginationService[User]()
		db, mock := setupTestDB(t)
		req := dtos.PaginationRequest{Page: 11, RowsPerPage: 10, OptimizeWithKey: "id"}
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users"`)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(150))

		idRows := sqlmock.NewRows([]string{"id"}).AddRow(101).AddRow(102)
		mock.ExpectQuery(`SELECT "id" FROM "users" ORDER BY "users"\."id" ASC LIMIT \$1 OFFSET \$2`).
			WithArgs(10, 100).
			WillReturnRows(idRows)

		dataRows := sqlmock.NewRows([]string{"id"}).AddRow(101).AddRow(102)
		mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" IN \(\$1,\$2\) ORDER BY "users"\."id" ASC`).
			WithArgs(101, 102).
			WillReturnRows(dataRows)
		_, err := service.Execute(db, req, nil, nil)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
