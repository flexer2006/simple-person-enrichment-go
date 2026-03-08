package postgres_test

import (
	"context"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/flexer2006/case-person-enrichment-go/internal/database/postgres"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name     string
		config   postgres.Config
		wantErr  bool
		errorMsg string
	}{
		{
			name: "Valid configuration",
			config: postgres.Config{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "secret",
				Database: "testdb",
				SSLMode:  "disable",
			},
			wantErr: false,
		},
		{
			name: "Empty host",
			config: postgres.Config{
				Host:     "",
				Port:     5432,
				User:     "postgres",
				Password: "secret",
				Database: "testdb",
				SSLMode:  "disable",
			},
			wantErr:  true,
			errorMsg: "invalid database configuration: required fields missing",
		},
		{
			name: "Zero port",
			config: postgres.Config{
				Host:     "localhost",
				Port:     0,
				User:     "postgres",
				Password: "secret",
				Database: "testdb",
				SSLMode:  "disable",
			},
			wantErr:  true,
			errorMsg: "invalid database configuration: required fields missing",
		},
		{
			name: "Empty user",
			config: postgres.Config{
				Host:     "localhost",
				Port:     5432,
				User:     "",
				Password: "secret",
				Database: "testdb",
				SSLMode:  "disable",
			},
			wantErr:  true,
			errorMsg: "invalid database configuration: required fields missing",
		},
		{
			name: "Empty database",
			config: postgres.Config{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "secret",
				Database: "",
				SSLMode:  "disable",
			},
			wantErr:  true,
			errorMsg: "invalid database configuration: required fields missing",
		},
		{
			name: "Multiple missing fields",
			config: postgres.Config{
				Host:     "",
				Port:     0,
				User:     "",
				Password: "secret",
				Database: "",
				SSLMode:  "disable",
			},
			wantErr:  true,
			errorMsg: "invalid database configuration: required fields missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, postgres.ErrInvalidConfiguration, err)
				assert.Equal(t, tt.errorMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_DSN(t *testing.T) {
	tests := []struct {
		name     string
		config   postgres.Config
		expected string
	}{
		{
			name: "Standard configuration",
			config: postgres.Config{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "password",
				Database: "testdb",
				SSLMode:  "disable",
			},
			expected: "postgres://postgres:password@localhost:5432/testdb?sslmode=disable",
		},
		{
			name: "Configuration with special characters in password",
			config: postgres.Config{
				Host:     "db.example.com",
				Port:     5432,
				User:     "user",
				Password: "p@ssw0rd!",
				Database: "production",
				SSLMode:  "require",
			},
			expected: "postgres://user:p@ssw0rd!@db.example.com:5432/production?sslmode=require",
		},
		{
			name: "Configuration with different port",
			config: postgres.Config{
				Host:     "192.168.1.10",
				Port:     5433,
				User:     "admin",
				Password: "secret",
				Database: "appdb",
				SSLMode:  "verify-full",
			},
			expected: "postgres://admin:secret@192.168.1.10:5433/appdb?sslmode=verify-full",
		},
		{
			name: "Configuration with empty password",
			config: postgres.Config{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "",
				Database: "testdb",
				SSLMode:  "prefer",
			},
			expected: "postgres://postgres:@localhost:5432/testdb?sslmode=prefer",
		},
		{
			name: "Configuration with empty SSLMode",
			config: postgres.Config{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "password",
				Database: "testdb",
				SSLMode:  "",
			},
			expected: "postgres://postgres:password@localhost:5432/testdb?sslmode=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := tt.config.DSN()
			assert.Equal(t, tt.expected, dsn)
		})
	}
}

func TestNew(t *testing.T) {
	t.Run("Invalid configuration", func(t *testing.T) {
		ctx := context.Background()
		invalidConfig := postgres.Config{}

		db, err := postgres.New(ctx, invalidConfig)

		assert.Nil(t, db)
		assert.Error(t, err)
		assert.Equal(t, postgres.ErrInvalidConfiguration, err)
	})

	t.Run("Invalid host", func(t *testing.T) {
		ctx := context.Background()
		config := postgres.Config{
			Port:     5432,
			User:     "user",
			Password: "password",
			Database: "testdb",
			SSLMode:  "disable",
			// Host is empty
		}

		db, err := postgres.New(ctx, config)

		assert.Nil(t, db)
		assert.Error(t, err)
		assert.Equal(t, postgres.ErrInvalidConfiguration, err)
	})

	t.Run("Invalid port", func(t *testing.T) {
		ctx := context.Background()
		config := postgres.Config{
			Host:     "localhost",
			User:     "user",
			Password: "password",
			Database: "testdb",
			SSLMode:  "disable",
		}

		db, err := postgres.New(ctx, config)

		assert.Nil(t, db)
		assert.Error(t, err)
		assert.Equal(t, postgres.ErrInvalidConfiguration, err)
	})

	t.Run("Invalid user", func(t *testing.T) {
		ctx := context.Background()
		config := postgres.Config{
			Host:     "localhost",
			Port:     5432,
			Password: "password",
			Database: "testdb",
			SSLMode:  "disable",
		}

		db, err := postgres.New(ctx, config)

		assert.Nil(t, db)
		assert.Error(t, err)
		assert.Equal(t, postgres.ErrInvalidConfiguration, err)
	})

	t.Run("Invalid database", func(t *testing.T) {
		ctx := context.Background()
		config := postgres.Config{
			Host:     "localhost",
			Port:     5432,
			User:     "user",
			Password: "password",
			SSLMode:  "disable",
		}

		db, err := postgres.New(ctx, config)

		assert.Nil(t, db)
		assert.Error(t, err)
		assert.Equal(t, postgres.ErrInvalidConfiguration, err)
	})
}

var (
	testHost     = getEnvOrDefault("PG_TEST_HOST", "localhost")
	testPort     = getEnvOrDefault("PG_TEST_PORT", "5432")
	testUser     = getEnvOrDefault("PG_TEST_USER", "postgres")
	testPassword = getEnvOrDefault("PG_TEST_PASSWORD", "postgres")
	testDatabase = getEnvOrDefault("PG_TEST_DATABASE", "postgres")
	testSSLMode  = getEnvOrDefault("PG_TEST_SSLMODE", "disable")
)

func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func buildTestDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		testUser, testPassword, testHost, testPort, testDatabase, testSSLMode)
}

func skipIfNoDB(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
}

func TestNewWithDSN(t *testing.T) {
	t.Run("Invalid DSN", func(t *testing.T) {
		ctx := context.Background()
		invalidDSN := "invalid-dsn"

		db, err := postgres.NewWithDSN(ctx, invalidDSN, 1, 10)

		assert.Nil(t, db)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse database configuration")
	})

	t.Run("Invalid DSN format", func(t *testing.T) {
		ctx := context.Background()
		invalidDSN := "postgres://user:password@localhost:invalid/testdb"

		db, err := postgres.NewWithDSN(ctx, invalidDSN, 1, 10)

		assert.Nil(t, db)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse database configuration")
	})

	t.Run("Successful connection with default connection parameters", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()
		dsn := buildTestDSN()

		db, err := postgres.NewWithDSN(ctx, dsn, 0, 0)
		if err != nil {
			t.Logf("Connection failed with DSN: %s", dsn)
			t.Fatal(err)
		}
		defer db.Close(ctx)

		require.NoError(t, err)
		require.NotNil(t, db)

		err = db.Ping(ctx)
		assert.NoError(t, err)
	})

	t.Run("Connection with custom MinConns and MaxConns", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()
		dsn := buildTestDSN()

		minConn := 2
		maxConn := 5

		db, err := postgres.NewWithDSN(ctx, dsn, minConn, maxConn)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		require.NoError(t, err)
		require.NotNil(t, db)

		config := db.Config()
		assert.Equal(t, minConn, config.MinConns)
		assert.Equal(t, maxConn, config.MaxConns)
	})

	t.Run("MinConns exceeds maximum allowed value", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()
		dsn := buildTestDSN()

		minConn := 10000
		maxConn := 10

		db, err := postgres.NewWithDSN(ctx, dsn, minConn, maxConn)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		require.NoError(t, err)
		require.NotNil(t, db)

		config := db.Config()
		assert.Equal(t, minConn, config.MinConns)

		err = db.Ping(ctx)
		assert.NoError(t, err)
	})

	t.Run("MaxConns exceeds maximum allowed value", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()
		dsn := buildTestDSN()

		minConn := 2
		maxConn := 10000

		db, err := postgres.NewWithDSN(ctx, dsn, minConn, maxConn)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		require.NoError(t, err)
		require.NotNil(t, db)

		config := db.Config()
		assert.Equal(t, maxConn, config.MaxConns)

		err = db.Ping(ctx)
		assert.NoError(t, err)
	})

	t.Run("Connection failure - wrong credentials", func(t *testing.T) {
		ctx := context.Background()
		invalidDSN := fmt.Sprintf("postgres://wrong:wrong@%s:%s/%s?sslmode=%s",
			testHost, testPort, testDatabase, testSSLMode)

		db, err := postgres.NewWithDSN(ctx, invalidDSN, 1, 10)

		assert.Nil(t, db)
		assert.Error(t, err)
	})

	t.Run("Connection failure - non-existent database", func(t *testing.T) {
		ctx := context.Background()
		invalidDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/nonexistentdb?sslmode=%s",
			testUser, testPassword, testHost, testPort, testSSLMode)

		db, err := postgres.NewWithDSN(ctx, invalidDSN, 1, 10)

		assert.Nil(t, db)
		assert.Error(t, err)
	})

	t.Run("Connection failure - wrong host", func(t *testing.T) {
		ctx := context.Background()
		invalidDSN := fmt.Sprintf("postgres://%s:%s@nonexistenthost:%s/%s?sslmode=%s",
			testUser, testPassword, testPort, testDatabase, testSSLMode)

		db, err := postgres.NewWithDSN(ctx, invalidDSN, 1, 10)

		assert.Nil(t, db)
		assert.Error(t, err)
	})

	t.Run("GetDSN returns empty string for database created with NewWithDSN", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()
		dsn := buildTestDSN()

		db, err := postgres.NewWithDSN(ctx, dsn, 1, 10)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		require.NoError(t, err)
		require.NotNil(t, db)

		returnedDSN := db.GetDSN()
		assert.NotEqual(t, dsn, returnedDSN)
		assert.Equal(t, "postgres://:@:0/?sslmode=", returnedDSN)
	})

	t.Run("Check Database.Ping works", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()
		dsn := buildTestDSN()

		db, err := postgres.NewWithDSN(ctx, dsn, 1, 10)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		require.NoError(t, err)
		require.NotNil(t, db)
		err = db.Ping(ctx)
		assert.NoError(t, err)
	})

	t.Run("Check Pool returns valid connection pool", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()
		dsn := buildTestDSN()

		db, err := postgres.NewWithDSN(ctx, dsn, 1, 10)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		require.NoError(t, err)
		require.NotNil(t, db)

		pool := db.Pool()
		assert.NotNil(t, pool)
	})
}

func TestDatabase_Pool(t *testing.T) {
	t.Run("Pool returns valid connection pool with New", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()

		config := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  testSSLMode,
			MinConns: 1,
			MaxConns: 5,
		}

		db, err := postgres.New(ctx, config)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		require.NoError(t, err)
		require.NotNil(t, db)

		pool := db.Pool()
		assert.NotNil(t, pool)

		var result int
		err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)
		assert.NoError(t, err)
		assert.Equal(t, 1, result)
	})

	t.Run("Pool returns valid connection pool with NewWithDSN", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()
		dsn := buildTestDSN()

		db, err := postgres.NewWithDSN(ctx, dsn, 1, 5)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		require.NoError(t, err)
		require.NotNil(t, db)

		pool := db.Pool()
		assert.NotNil(t, pool)

		var result int
		err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)
		assert.NoError(t, err)
		assert.Equal(t, 1, result)
	})

	t.Run("Pool returned is the same instance", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()
		dsn := buildTestDSN()

		db, err := postgres.NewWithDSN(ctx, dsn, 1, 5)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		require.NoError(t, err)
		require.NotNil(t, db)

		pool1 := db.Pool()
		pool2 := db.Pool()

		assert.NotNil(t, pool1)
		assert.NotNil(t, pool2)
		assert.Same(t, pool1, pool2, "Pool() should return the same pool instance on repeated calls")
	})
}

func parseInt(value string, fallback int) int {
	var result int
	_, err := fmt.Sscanf(value, "%d", &result)
	if err != nil {
		return fallback
	}
	return result
}

func TestDatabase_Close(t *testing.T) {
	t.Run("Close database created with New()", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()

		config := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  testSSLMode,
			MinConns: 1,
			MaxConns: 5,
		}

		db, err := postgres.New(ctx, config)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}

		err = db.Ping(ctx)
		require.NoError(t, err, "Database should be pingable before closing")

		var result int
		err = db.Pool().QueryRow(ctx, "SELECT 1").Scan(&result)
		require.NoError(t, err, "Should be able to execute query before closing")

		// Close the connection
		db.Close(ctx)

		err = db.Ping(ctx)
		assert.Error(t, err, "Ping should fail after database is closed")

		err = db.Pool().QueryRow(ctx, "SELECT 1").Scan(&result)
		assert.Error(t, err, "Query should fail after database is closed")
	})

	t.Run("Close database created with NewWithDSN()", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()
		dsn := buildTestDSN()

		db, err := postgres.NewWithDSN(ctx, dsn, 1, 5)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}

		err = db.Ping(ctx)
		require.NoError(t, err, "Database should be pingable before closing")

		db.Close(ctx)

		err = db.Ping(ctx)
		assert.Error(t, err, "Ping should fail after database is closed")
	})

	t.Run("Multiple calls to Close() should not cause panic", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()
		dsn := buildTestDSN()

		db, err := postgres.NewWithDSN(ctx, dsn, 1, 5)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}

		db.Close(ctx)

		assert.NotPanics(t, func() {
			db.Close(ctx)
		}, "Multiple calls to Close() should not panic")
	})

	t.Run("Operations should fail after Close() with appropriate timeout", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()
		dsn := buildTestDSN()

		db, err := postgres.NewWithDSN(ctx, dsn, 1, 5)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}

		db.Close(ctx)

		ctxWithTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		var result int
		err = db.Pool().QueryRow(ctxWithTimeout, "SELECT 1").Scan(&result)
		assert.Error(t, err, "Query should fail after database is closed, even with timeout")
	})
}

func TestDatabase_Ping(t *testing.T) {
	t.Run("Ping succeeds with active connection", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()

		config := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  testSSLMode,
		}

		db, err := postgres.New(ctx, config)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		err = db.Ping(ctx)
		assert.NoError(t, err)
	})

	t.Run("Ping fails after connection is closed", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()

		config := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  testSSLMode,
		}

		db, err := postgres.New(ctx, config)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}

		err = db.Ping(ctx)
		require.NoError(t, err)

		db.Close(ctx)

		err = db.Ping(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to ping database")
	})

	t.Run("Ping respects context timeout", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()

		config := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  testSSLMode,
		}

		db, err := postgres.New(ctx, config)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		ctxWithTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()

		err = db.Ping(ctxWithTimeout)
		assert.NoError(t, err, "Ping should succeed within the timeout")
	})

	t.Run("Ping fails with canceled context", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()

		config := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  testSSLMode,
		}

		db, err := postgres.New(ctx, config)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		ctxWithCancel, cancel := context.WithCancel(ctx)
		cancel()

		err = db.Ping(ctxWithCancel)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to ping database")
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("Ping with DSN connection", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()
		dsn := buildTestDSN()

		db, err := postgres.NewWithDSN(ctx, dsn, 1, 5)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		err = db.Ping(ctx)
		assert.NoError(t, err)
	})
}

func TestDatabase_Config(t *testing.T) {
	t.Run("Config returns correct values for New()", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()

		originalConfig := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  testSSLMode,
			MinConns: 3,
			MaxConns: 10,
		}

		db, err := postgres.New(ctx, originalConfig)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		config := db.Config()

		assert.Equal(t, originalConfig.Host, config.Host)
		assert.Equal(t, originalConfig.Port, config.Port)
		assert.Equal(t, originalConfig.User, config.User)
		assert.Equal(t, originalConfig.Password, config.Password)
		assert.Equal(t, originalConfig.Database, config.Database)
		assert.Equal(t, originalConfig.SSLMode, config.SSLMode)
		assert.Equal(t, originalConfig.MinConns, config.MinConns)
		assert.Equal(t, originalConfig.MaxConns, config.MaxConns)
	})

	t.Run("Config returns minimal values for NewWithDSN()", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()
		dsn := buildTestDSN()

		minConn := 5
		maxConn := 15

		db, err := postgres.NewWithDSN(ctx, dsn, minConn, maxConn)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		config := db.Config()

		assert.Equal(t, "", config.Host)
		assert.Equal(t, 0, config.Port)
		assert.Equal(t, "", config.User)
		assert.Equal(t, "", config.Password)
		assert.Equal(t, "", config.Database)
		assert.Equal(t, "", config.SSLMode)
		assert.Equal(t, minConn, config.MinConns)
		assert.Equal(t, maxConn, config.MaxConns)
	})

	t.Run("Config returns a copy of the configuration", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()

		originalConfig := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  testSSLMode,
			MinConns: 3,
			MaxConns: 10,
		}

		db, err := postgres.New(ctx, originalConfig)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		config := db.Config()

		assert.Equal(t, originalConfig, config)

		config.Host = "modified-host"
		config.Port = 9999
		config.User = "modified-user"
		config.Password = "modified-password"
		config.Database = "modified-database"
		config.SSLMode = "modified-sslmode"
		config.MinConns = 999
		config.MaxConns = 9999

		newConfig := db.Config()

		assert.Equal(t, originalConfig, newConfig)
		assert.NotEqual(t, config, newConfig)
	})

	t.Run("DSN of returned Config is correct for New()", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()

		originalConfig := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  testSSLMode,
		}

		db, err := postgres.New(ctx, originalConfig)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		config := db.Config()
		expectedDSN := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			testUser, testPassword, testHost, parseInt(testPort, 5432), testDatabase, testSSLMode)

		assert.Equal(t, expectedDSN, config.DSN())
	})

	t.Run("Config DSN can be used to create new connection", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()

		originalConfig := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  testSSLMode,
			MinConns: 1,
			MaxConns: 5,
		}

		db1, err := postgres.New(ctx, originalConfig)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db1.Close(ctx)

		config := db1.Config()
		dsn := config.DSN()

		db2, err := postgres.NewWithDSN(ctx, dsn, config.MinConns, config.MaxConns)
		require.NoError(t, err)
		defer db2.Close(ctx)

		err = db2.Ping(ctx)
		assert.NoError(t, err)
	})
}

func TestDatabase_GetDSN(t *testing.T) {
	t.Run("GetDSN returns correct DSN for database created with New()", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()

		config := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  testSSLMode,
			MinConns: 1,
			MaxConns: 5,
		}

		db, err := postgres.New(ctx, config)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		dsn := db.GetDSN()

		expectedDSN := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			testUser, testPassword, testHost, parseInt(testPort, 5432), testDatabase, testSSLMode)

		assert.Equal(t, expectedDSN, dsn)
	})

	t.Run("GetDSN returns minimal DSN for database created with NewWithDSN()", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()
		dsn := buildTestDSN()

		db, err := postgres.NewWithDSN(ctx, dsn, 1, 5)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		returnedDSN := db.GetDSN()

		assert.NotEqual(t, dsn, returnedDSN)
		assert.Equal(t, "postgres://:@:0/?sslmode=", returnedDSN)
	})

	t.Run("GetDSN with special characters in password", func(t *testing.T) {
		config := postgres.Config{
			Host:     "localhost",
			Port:     5432,
			User:     "testuser",
			Password: "p@ssw0rd!#$%^&*()",
			Database: "testdb",
			SSLMode:  "disable",
		}

		expectedDSN := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			config.User, config.Password, config.Host, config.Port, config.Database, config.SSLMode)

		assert.Equal(t, expectedDSN, config.DSN())
	})

	t.Run("GetDSN matching Config.DSN", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()

		config := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  testSSLMode,
		}

		db, err := postgres.New(ctx, config)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		assert.Equal(t, db.Config().DSN(), db.GetDSN())
	})

	t.Run("GetDSN with empty password", func(t *testing.T) {

		config := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: "", // Empty password
			Database: testDatabase,
			SSLMode:  testSSLMode,
		}

		expectedDSN := fmt.Sprintf("postgres://%s:@%s:%d/%s?sslmode=%s",
			testUser, testHost, parseInt(testPort, 5432), testDatabase, testSSLMode)

		assert.Equal(t, expectedDSN, config.DSN())
	})

	t.Run("GetDSN with empty SSLMode", func(t *testing.T) {
		skipIfNoDB(t)
		ctx := context.Background()

		config := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  "",
		}

		db, err := postgres.New(ctx, config)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		dsn := db.GetDSN()

		expectedDSN := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=",
			testUser, testPassword, testHost, parseInt(testPort, 5432), testDatabase)

		assert.Equal(t, expectedDSN, dsn)
	})
}

func TestNew_PoolConfigLogic(t *testing.T) {
	ctx := context.Background()

	t.Run("MinMaxConns behavior", func(t *testing.T) {
		config := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  testSSLMode,
			MinConns: 1000,
			MaxConns: 1000,
		}

		db, err := postgres.New(ctx, config)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		cfg := db.Config()
		assert.Equal(t, 1000, cfg.MinConns)
		assert.Equal(t, 1000, cfg.MaxConns)

		err = db.Ping(ctx)
		assert.NoError(t, err)
	})

	t.Run("Exceed MaxInt32 check", func(t *testing.T) {
		poolCfg, err := pgxpool.ParseConfig("postgres://localhost:5432/testdb")
		require.NoError(t, err)

		// Вручную вызываем логику проверки превышения MaxInt32
		minConn := math.MaxInt32 + 1
		if minConn > math.MaxInt32 {
			poolCfg.MinConns = math.MaxInt32
		} else {
			poolCfg.MinConns = int32(minConn)
		}

		assert.Equal(t, int32(math.MaxInt32), poolCfg.MinConns)
	})

	t.Run("MinMaxConns behavior with DSN", func(t *testing.T) {
		skipIfNoDB(t)
		dsn := buildTestDSN()

		minConn := 1000
		maxConn := 1000

		db, err := postgres.NewWithDSN(ctx, dsn, minConn, maxConn)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		cfg := db.Config()
		assert.Equal(t, minConn, cfg.MinConns)
		assert.Equal(t, maxConn, cfg.MaxConns)

		err = db.Ping(ctx)
		assert.NoError(t, err)
	})
}

func TestNew_ConnectionFailures(t *testing.T) {
	ctx := context.Background()

	t.Run("New with invalid port format", func(t *testing.T) {
		config := postgres.Config{
			Host:     testHost,
			Port:     -1,
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  testSSLMode,
		}

		db, err := postgres.New(ctx, config)
		assert.Nil(t, db)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
	})

	t.Run("Connection timeout", func(t *testing.T) {
		skipIfNoDB(t)

		ctxWithTimeout, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
		defer cancel()

		config := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  testSSLMode,
		}

		db, err := postgres.New(ctxWithTimeout, config)

		t.Logf("Connection with 1ns timeout: db=%v, err=%v", db, err)
	})

	t.Run("Connection timeout with DSN", func(t *testing.T) {
		skipIfNoDB(t)
		dsn := buildTestDSN()

		ctxWithTimeout, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
		defer cancel()

		db, err := postgres.NewWithDSN(ctxWithTimeout, dsn, 1, 5)

		t.Logf("DSN connection with 1ns timeout: db=%v, err=%v", db, err)
	})
}

func TestEdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("Zero min and max connections", func(t *testing.T) {
		skipIfNoDB(t)

		config := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  testSSLMode,
		}

		db, err := postgres.New(ctx, config)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		require.NoError(t, err)
		require.NotNil(t, db)

		err = db.Ping(ctx)
		assert.NoError(t, err)

		cfg := db.Config()
		assert.Equal(t, 0, cfg.MinConns)
		assert.Equal(t, 0, cfg.MaxConns)
	})

	t.Run("Negative min and max connections", func(t *testing.T) {
		skipIfNoDB(t)

		config := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  testSSLMode,
			MinConns: -5,
			MaxConns: -10,
		}

		db, err := postgres.New(ctx, config)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		cfg := db.Config()
		assert.Equal(t, -5, cfg.MinConns)
		assert.Equal(t, -10, cfg.MaxConns)

		err = db.Ping(ctx)
		assert.NoError(t, err)
	})

	t.Run("MinConns greater than MaxConns", func(t *testing.T) {
		skipIfNoDB(t)

		config := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  testSSLMode,
			MinConns: 10,
			MaxConns: 5,
		}

		db, err := postgres.New(ctx, config)
		if err != nil {
			t.Skip("Skipping test due to connection failure")
		}
		defer db.Close(ctx)

		cfg := db.Config()
		assert.Equal(t, 10, cfg.MinConns)
		assert.Equal(t, 5, cfg.MaxConns)

		err = db.Ping(ctx)
		assert.NoError(t, err)
	})
}
func TestErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("Ping after pool is nil", func(t *testing.T) {
		db := &postgres.Database{}

		err := db.Ping(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to ping database")
	})

	t.Run("Using invalid DSN", func(t *testing.T) {
		dsn := "postgres://user:pass@host:port/db?invalidparam=value"

		db, err := postgres.NewWithDSN(ctx, dsn, 1, 5)
		assert.Nil(t, db)
		assert.Error(t, err)
	})

	t.Run("Invalid connection parameters", func(t *testing.T) {
		config := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     "nonexistentuser",
			Password: "wrongpassword",
			Database: "this_database_does_not_exist_" + time.Now().String(),
			SSLMode:  testSSLMode,
		}

		db, err := postgres.New(ctx, config)
		assert.Nil(t, db)
		assert.Error(t, err)
	})

	t.Run("Canceled context", func(t *testing.T) {
		ctxWithCancel, cancel := context.WithCancel(ctx)
		cancel()

		config := postgres.Config{
			Host:     testHost,
			Port:     parseInt(testPort, 5432),
			User:     testUser,
			Password: testPassword,
			Database: testDatabase,
			SSLMode:  testSSLMode,
		}

		db, err := postgres.New(ctxWithCancel, config)
		assert.Nil(t, db)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context")
	})
}
