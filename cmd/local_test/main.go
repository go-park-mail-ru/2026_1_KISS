package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	nbpg "github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/repository/postgres"
	nbusecase "github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/database"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/container"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/runner_service"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/session_repository"
)

func createBlocks(ctx context.Context, cfg *config.Config) {
	db, err := database.Connect(cfg.Database.DSN())
	if err != nil {
		panic(err)
	}
	notebookRepo := nbpg.NewNotebookRepository(db)
	blockRepo := nbpg.NewBlockRepository(db)
	userID := int64(1)
	notebookID := int64(1)
	notebookServ := nbusecase.New(notebookRepo, blockRepo)

	notebook, err := notebookServ.GetByID(ctx, userID, notebookID)
	if err != nil {
		panic(err)
	}

	codes := []string{"a = 3", "b = 4", "print(a + b)"}
	for i, code := range codes {
		ind := i + 1
		block := &domain.Block{
			int64(ind + 1), notebook.ID, "code", "python",
			code, ind, nil,
			[]domain.BlockOutput{}, time.Now(), time.Now(),
		}
		if _, err := notebookServ.AddBlock(ctx, userID, notebook.ID, block); err != nil {
			panic(err)
		}
	}

}

func main() {
	ctx := context.Background()
	cfg := config.Load()
	runnerManager, err := container.NewManager(cfg.Runner)
	if err != nil {
		panic(err)
	}
	sessionRepo := session_repository.NewExecutionSessionRepository()

	//userID := int64(1)
	notebookID := int64(1)
	db, err := database.Connect(cfg.Database.DSN())
	if err != nil {
		panic(err)
	}
	notebookRepo := nbpg.NewNotebookRepository(db)
	blockRepo := nbpg.NewBlockRepository(db)

	runnerServ := runner_service.NewRunnerService(runnerManager, sessionRepo, notebookRepo, blockRepo)
	if err := runnerServ.StartSession(ctx, notebookID); err != nil {
		panic(err)
	}

	{
		execResult, err := runnerServ.ExecuteBlock(ctx, notebookID, 0)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Executing result: \n%s\n", execResult)
	}
	{
		_, err := runnerServ.ExecuteFromPosition(ctx, notebookID, 0)
		if err != nil {
			panic(err)
		}
		//for _, v := range execResult {
		//	fmt.Printf("Executing result: \n%s\n", v)
		//}
		execResult, err := runnerServ.ExecuteBlock(ctx, notebookID, 3)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Executing result: \n%s\n", execResult)
	}

	if err := runnerServ.StopSession(ctx, notebookID); err != nil {
		panic(err)
	}

}

type ExecuteRequest struct {
	Code    string  `json:"code"`
	Timeout float64 `json:"timeout,omitempty"` // Optional, defaults to 15
}

type ExecuteResponse struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Result string `json:"result"`
}

func main2() {
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
	{
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
	}

	// Execute request - Example 1: Simple print
	{
		executeRequest1 := ExecuteRequest{
			Code:    "x = 10\nprint(f'x = {x + 3}')",
			Timeout: 15,
		}
		jsonData, err := json.Marshal(executeRequest1)
		if err != nil {
			log.Fatal(err)
		}
		codeResp, err := http.Post(baseURL+"/execute", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Printf("execute request failed: %v", err)
		} else {
			defer codeResp.Body.Close()

			body, err := io.ReadAll(codeResp.Body)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("Execute Response Status: %d\n", codeResp.StatusCode)
			fmt.Printf("Execute Response Body: %s\n", string(body))

			// Parse the response
			var executeResp ExecuteResponse
			if err := json.Unmarshal(body, &executeResp); err == nil {
				fmt.Printf("Stdout: %s\n", executeResp.Stdout)
				fmt.Printf("Stderr: %s\n", executeResp.Stderr)
				fmt.Printf("Result: %s\n", executeResp.Result)
			}
		}
	}

	// Example 2: Execute code that returns a value
	{
		executeRequest2 := ExecuteRequest{
			Code:    "result = 42\nresult",
			Timeout: 10,
		}

		jsonData2, err := json.Marshal(executeRequest2)
		if err != nil {
			log.Fatal(err)
		}

		codeResp2, err := http.Post(baseURL+"/execute", "application/json", bytes.NewBuffer(jsonData2))
		if err != nil {
			log.Printf("execute request 2 failed: %v", err)
		} else {
			defer codeResp2.Body.Close()

			body, err := io.ReadAll(codeResp2.Body)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("\nExecute Response 2: %s\n", string(body))
		}
	}

	if err := runnerManager.StopSession(ctx, sessionID); err != nil {
		fmt.Printf("failed to stop session: %v", err)
	}
	runnerManager.CleanupSessions(ctx)
}
