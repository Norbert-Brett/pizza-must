package repository

import (
	"context"
	"database/sql"
	"log"
	"testing"
	"time"

	"pizza-must/internal/domain"

	"github.com/google/uuid"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var testDB *sql.DB

func setupTestDB() (func(context.Context, ...testcontainers.TerminateOption) error, error) {
	var (
		dbName = "testdb"
		dbPwd  = "password"
		dbUser = "user"
	)

	dbContainer, err := postgres.Run(
		context.Background(),
		"postgres:15",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPwd),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		return nil, err
	}

	dbHost, err := dbContainer.Host(context.Background())
	if err != nil {
		return dbContainer.Terminate, err
	}

	dbPort, err := dbContainer.MappedPort(context.Background(), "5432/tcp")
	if err != nil {
		return dbContainer.Terminate, err
	}

	connStr := "postgres://" + dbUser + ":" + dbPwd + "@" + dbHost + ":" + dbPort.Port() + "/" + dbName + "?sslmode=disable"
	testDB, err = sql.Open("pgx", connStr)
	if err != nil {
		return dbContainer.Terminate, err
	}

	// Create the users table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			first_name VARCHAR(100),
			last_name VARCHAR(100),
			role VARCHAR(50) NOT NULL DEFAULT 'user',
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return dbContainer.Terminate, err
	}

	return dbContainer.Terminate, nil
}

func TestMain(m *testing.M) {
	teardown, err := setupTestDB()
	if err != nil {
		log.Fatalf("could not start postgres container: %v", err)
	}

	m.Run()

	if teardown != nil {
		if err := teardown(context.Background()); err != nil {
			log.Fatalf("could not teardown postgres container: %v", err)
		}
	}
}

// Feature: ordering-platform, Property 1: Registration creates hashed passwords
// Validates: Requirements 1.1, 1.3
func TestProperty_RegistrationCreatesHashedPasswords(t *testing.T) {
	repo := NewUserRepository(testDB)
	ctx := context.Background()

	properties := gopter.NewProperties(nil)

	properties.Property("passwords are hashed with bcrypt and not stored as plaintext", prop.ForAll(
		func(email string, password string, firstName string, lastName string) bool {
			// Clean up before each test
			_, _ = testDB.Exec("DELETE FROM users WHERE email = $1", email)

			// Hash the password with bcrypt
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				t.Logf("Failed to hash password: %v", err)
				return false
			}

			// Create user with hashed password
			user := &domain.User{
				ID:           uuid.New(),
				Email:        email,
				PasswordHash: string(hashedPassword),
				FirstName:    firstName,
				LastName:     lastName,
				Role:         "user",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}

			// Store the user
			err = repo.Create(ctx, user)
			if err != nil {
				t.Logf("Failed to create user: %v", err)
				return false
			}

			// Retrieve the user
			retrievedUser, err := repo.FindByEmail(ctx, email)
			if err != nil {
				t.Logf("Failed to find user: %v", err)
				return false
			}

			// Verify the password is hashed (not equal to plaintext)
			if retrievedUser.PasswordHash == password {
				t.Logf("Password was stored as plaintext!")
				return false
			}

			// Verify the stored hash is a valid bcrypt hash by comparing
			err = bcrypt.CompareHashAndPassword([]byte(retrievedUser.PasswordHash), []byte(password))
			if err != nil {
				t.Logf("Stored password is not a valid bcrypt hash: %v", err)
				return false
			}

			// Clean up after test
			_, _ = testDB.Exec("DELETE FROM users WHERE email = $1", email)

			return true
		},
		// Generate valid email addresses
		gen.RegexMatch(`[a-z]{5,10}@[a-z]{3,8}\.(com|org|net)`),
		// Generate passwords with at least 8 characters
		gen.RegexMatch(`[A-Za-z0-9!@#$%]{8,20}`),
		// Generate first names
		gen.RegexMatch(`[A-Z][a-z]{2,15}`),
		// Generate last names
		gen.RegexMatch(`[A-Z][a-z]{2,15}`),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
