package appcontext

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/RoyceAzure/lab/authcenter/internal/config"
	"github.com/RoyceAzure/lab/authcenter/internal/infra/auth/google_auth"
	"github.com/RoyceAzure/lab/authcenter/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/authcenter/internal/infra/repository/db/sqlc"
	"github.com/RoyceAzure/lab/authcenter/internal/service"
	"github.com/RoyceAzure/lab/authcenter/internal/util"
	"github.com/RoyceAzure/rj/api/token"
	util_http "github.com/RoyceAzure/rj/util/rj_http"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ApplicationContext struct {
	HttpClient         *util_http.Client
	DbConn             *pgxpool.Pool
	DbDao              db.IStore
	Cf                 *config.Config
	TokenMaker         token.Maker[uuid.UUID]
	GoogleAuthVerifier google_auth.IAuthVerifier
	SessionService     service.ISessionService
	UserService        service.IUserService
	AuthService        service.IAuthService
	MailService        service.IMailService
}

func NewApplicationContext(cf *config.Config) (*ApplicationContext, error) {
	app := ApplicationContext{
		Cf: cf,
	}
	v := reflect.ValueOf(*cf)
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		fieldName := t.Field(i).Name
		fieldValue := v.Field(i).Interface()
		fmt.Printf("  \"%s\": \"%v\",\n", fieldName, fieldValue)
	}
	err := app.Init()

	if err != nil {
		return nil, err
	}

	return &app, nil
}

func (app *ApplicationContext) Init() error {
	app.setUpHttpClient()
	err := app.setUpHttpClient()
	if err != nil {
		return err
	}
	err = app.setUpdbConn()
	if err != nil {
		return err
	}
	err = app.setUpdbDao()
	if err != nil {
		return err
	}

	err = app.setUpSessionService()
	if err != nil {
		return err
	}

	err = app.setUpUserService()
	if err != nil {
		return err
	}

	err = app.setUpMailService()
	if err != nil {
		return err
	}

	err = app.setTokenMaker()
	if err != nil {
		return err
	}

	err = app.setGoogleVerifier()
	if err != nil {
		return err
	}

	err = app.setUpAuthService()
	if err != nil {
		return err
	}
	// err = app.dbInit()
	// if err != nil {
	// 	return err
	// }

	//強制清空所有user session  for server意外關閉情況
	log.Printf("force cleanning all user session...")
	app.SessionService.ForceClearAllSessions(context.TODO())
	log.Printf("force cleanning all user session successed")

	return nil
}

func (app *ApplicationContext) setUpHttpClient() error {
	log.Printf("Start setup HTTP client")
	app.HttpClient = util_http.NewHttpClient(util_http.WithTimeout(time.Second * 30))
	log.Printf("Finish setup HTTP client")
	return nil
}

func (app *ApplicationContext) setUpdbConn() error {
	log.Printf("Start setup database connection")
	conn, err := pgxpool.New(context.Background(),
		fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", app.Cf.DbUser, app.Cf.DbPas, app.Cf.DbHost, app.Cf.DbPort, app.Cf.DbName))
	if err != nil {
		return err
	}
	app.DbConn = conn
	log.Printf("Finish setup database connection")
	return nil
}

func (app *ApplicationContext) setUpdbDao() error {
	log.Printf("Start setup database DAO")
	app.DbDao = db.NewStore(app.DbConn)
	log.Printf("Finish setup database DAO")
	return nil
}

func (app *ApplicationContext) setUpSessionService() error {
	log.Printf("Start setup session service")
	app.SessionService = service.NewSessionService(app.DbDao)
	log.Printf("Finish setup session service")
	return nil
}

func (app *ApplicationContext) setUpUserService() error {
	log.Printf("Start setup user service")
	app.UserService = service.NewUserService(app.DbDao)
	log.Printf("Finish setup user service")
	return nil
}

func (app *ApplicationContext) setUpMailService() error {
	log.Printf("Start setup mail service")
	app.MailService = service.NewMailService(app.Cf.ModulerName, app.Cf.EmailAccount, app.Cf.SmtpAuthKey)
	log.Printf("Finish setup mail service")
	return nil
}

func (app *ApplicationContext) setGoogleVerifier() error {
	log.Printf("Start setup googleVerifier")
	app.GoogleAuthVerifier = google_auth.NewGoogleAuthVerifier(app.Cf.GoogleClientID)
	log.Printf("Finish setup googleVerifier")
	return nil
}

func (app *ApplicationContext) setUpAuthService() error {
	log.Printf("Start setup auth service")
	app.AuthService = service.NewAuthService(app.DbDao, app.UserService, app.SessionService, app.MailService, app.TokenMaker, app.GoogleAuthVerifier)
	log.Printf("Finish setup auth service")
	return nil
}

func (app *ApplicationContext) setTokenMaker() error {
	log.Printf("Start setup token maker")

	tokenMaker, err := token.NewPasetoMaker[uuid.UUID](app.Cf.AuthTokenKey)
	if err != nil {
		log.Fatalf("無法創建 token maker: %v", err)
	}

	app.TokenMaker = tokenMaker
	log.Printf("Finish setup token maker")
	return nil
}

func (app *ApplicationContext) Shutdown(ctx context.Context) error {
	log.Printf("Start application shutdown")

	done := make(chan error)
	go func() {
		defer close(done)

		// if app.SchedulerService != nil {
		// 	log.Printf("Stopping scheduler service...")
		// 	if err := app.SchedulerService.Stop(); err != nil {
		// 		//有錯誤不結束流程
		// 		log.Printf("scheduler service shutdown error: %v", err)
		// 	}
		// }

		//強制清空所有user session
		log.Printf("force cleanning all user session...")
		app.SessionService.ForceClearAllSessions(ctx)
		log.Printf("force cleanning all user session successed")

		// 關閉 DB
		if app.DbConn != nil {
			log.Printf("Closing database connection...")
			app.DbConn.Close()
		}

		// 關閉 logger
		log.Printf("Shutting down logger...")
		// 實作 logger 關閉邏輯

		log.Printf("Application shutdown complete")
		done <- nil
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout: %v", ctx.Err())
	}
}

func runDBMigration(migrationURL string, dbSource string) error {
	migrateion, err := migrate.New(migrationURL, dbSource)
	if err != nil {
		return err
	}

	return migrateion.Up()
}

func trimLeadingSlash(path string) string {
	if len(path) > 0 && path[0] == '/' {
		return path[1:]
	}
	return path
}

// db migration and db seed data
func (app *ApplicationContext) dbInit() error {
	log.Printf("Start setup db init")

	migrateUrl := trimLeadingSlash("internal/infra/repository/db/migrations")
	err := runDBMigration(
		fmt.Sprintf("file://%s", migrateUrl),
		fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", app.Cf.DbUser, app.Cf.DbPas, app.Cf.DbHost, app.Cf.DbPort, app.Cf.DbName),
	)

	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	ctx := context.Background()
	//db data init
	perCf, err := config.LoadPermissionConfig(fmt.Sprintf("%s/docs/permission.yaml", util.GetProjectRoot("github.com/RoyceAzure/lab/authcenter")))
	if err != nil {
		return err
	}

	var setPermission = func(repo *sqlc.Queries) error {
		for _, permission := range perCf.Permissions {
			err := repo.CreatePermissionIfNotExists(ctx, sqlc.CreatePermissionIfNotExistsParams{
				ID:       permission.ID,
				Name:     permission.Name,
				Resource: permission.Resource,
				Actions:  permission.Actions,
			})
			if err != nil {
				return err
			}
		}
		return nil
	}

	var setRoleAndRolePermission = func(repo *sqlc.Queries) error {
		for _, rolePermission := range perCf.RolePermission {
			//create role
			err := repo.CreateRoleIfNotExists(ctx, sqlc.CreateRoleIfNotExistsParams{
				ID:   rolePermission.RoleID,
				Name: rolePermission.Name,
			})
			if err != nil {
				return err
			}

			//create role permission
			for _, permissionID := range rolePermission.Permissions {
				err = repo.CreateRolePermissionIfNotExists(ctx, sqlc.CreateRolePermissionIfNotExistsParams{
					RoleID:       rolePermission.RoleID,
					PermissionID: permissionID,
				})
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	var setUserAndRole = func(repo *sqlc.Queries) error {
		//create super admin user
		// name := "royce"
		// adminID := pgutil.UUIDToPgUUIDV5(uuid.New())
		// err := repo.CreateUserIfNotExists(ctx, sqlc.CreateUserIfNotExistsParams{
		// 	ID:        adminID,
		// 	Email:     "roycewnag@gmail.com",
		// 	Name:      pgutil.StringToPgTextV5(&name),
		// 	CreatedAt: time.Now().UTC(),
		// 	IsAdmin:   true,
		// 	IsActive:  true,
		// })
		// if err != nil {
		// 	return err
		// }

		// err = repo.CreateUserRoleIfNotExists(ctx, sqlc.CreateUserRoleIfNotExistsParams{
		// 	UserID: adminID,
		// 	RoleID: 1,
		// })
		// if err != nil {
		// 	return err
		// }

		return nil
	}

	app.DbDao.ExecMultiTx(ctx, []func(*sqlc.Queries) error{setPermission, setRoleAndRolePermission, setUserAndRole})

	log.Printf("Finish setup db init")
	return nil
}
