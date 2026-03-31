package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	nbpg "github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/repository/postgres"
	nbusecase "github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/database"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/container"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()
	db, err := database.Connect(cfg.Database.DSN())
	if err != nil {
		panic(err)
	}
	//
	//rdb := redisv9.NewClient(&redisv9.Options{
	//	Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
	//	Password: cfg.Redis.Password,
	//	DB:       0,
	//})

	//userRepo := authpg.NewUserRepository(db)
	//sessionRepo := authredis.NewSessionRepository(rdb)
	notebookRepo := nbpg.NewNotebookRepository(db)
	blockRepo := nbpg.NewBlockRepository(db)
	//authUC := authusecase.New(userRepo, sessionRepo, cfg.Auth.SessionTTL)

	// User
	//username := "Kate"
	//email := "test@mail.ru"
	//password := "password"
	//user, err := authUC.Register(ctx, username, email, password)
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Printf("User registered with ID %s\n", user.ID)

	userID := int64(1)

	notebookServ := nbusecase.New(notebookRepo, blockRepo)

	notebook, err := notebookServ.Create(ctx, userID, "title_test")
	if err != nil {
		panic(err)
	}

	//newBlock := &domain.Block{
	//	1, 1, "code", "python",
	//	"print('Hello')", 0, nil,
	//	[]domain.BlockOutput{}, time.Now(), time.Now(),
	//}
	//block, err := notebookServ.AddBlock(ctx, userID, notebook.ID, newBlock)
	//if err != nil {
	//	panic(err)
	//}
	block, err := notebookServ.GetByID(ctx, userID, notebook.ID)
	fmt.Println(block)

	runnerManager, err := container.NewManager(cfg.Runner)
	if err != nil {
		_ = db.Close()
		panic(err)
	}
	sessionID := "user-123-session-456"
	containerIP, err := runnerManager.StartSession(ctx, sessionID)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Address: %s\n", containerIP)

	//adr2, err := runnerManager.GetContainerAddress(ctx, "s-3")
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Printf("Adr2: %s\n", adr2)

	baseURL := "http://" + containerIP

	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		log.Printf("health check failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("runner is healthy")

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Body: %s\n", string(body))
	}

	if err := runnerManager.StopSession(ctx, sessionID); err != nil {
		fmt.Printf("failed to stop session: %v", err)
	}
	runnerManager.CleanupSessions(ctx)
}
