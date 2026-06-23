package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	serverURL  = "http://localhost:3223"
	wsURL      = "ws://localhost:3223"
	numRooms   = 10
	numPlayers = 8
)

type ReqMsg struct {
	MsgType string
	Data    string
}

type RespMsg struct {
	MsgType string
	RoomID  string
	Data    json.RawMessage
}

type RoomResult struct {
	roomIndex int
	connected int
	failed    int
	gameEnded bool
	errors    []string
}

func registerAndGetToken(username string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/register?username=%s", serverURL, url.QueryEscape(username)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("register failed (status %d): %s", resp.StatusCode, string(body))
	}
	return string(body), nil
}

func runRoom(roomIndex int, wg *sync.WaitGroup, results chan<- RoomResult) {
	defer wg.Done()

	roomID := fmt.Sprintf("loadtest-room-%d", roomIndex)
	result := RoomResult{roomIndex: roomIndex}
	var mu sync.Mutex
	var playerWg sync.WaitGroup
	gameEndedCount := 0

	// register all players first, before any WS connections
	tokens := make([]string, 0, numPlayers)
	for i := 0; i < numPlayers; i++ {
		username := fmt.Sprintf("r%d_u%d_%d", roomIndex, i, time.Now().UnixNano())
		token, err := registerAndGetToken(username)
		if err != nil {
			errMsg := fmt.Sprintf("Player %d registration failed: %v", i, err)
			mu.Lock()
			result.failed++
			result.errors = append(result.errors, errMsg)
			mu.Unlock()
			continue
		}
		tokens = append(tokens, token)
	}

	if len(tokens) == 0 {
		results <- result
		return
	}

	// connect all players
	for i, token := range tokens {
		playerWg.Add(1)
		isHost := i == 0

		go func(playerIndex int, token string, isHost bool) {
			defer playerWg.Done()

			wsAddr := fmt.Sprintf("%s/?room=%s&token=%s", wsURL, roomID, token)
			conn, _, err := websocket.DefaultDialer.Dial(wsAddr, nil)
			if err != nil {
				errMsg := fmt.Sprintf("Player %d ws dial failed: %v", playerIndex, err)
				mu.Lock()
				result.failed++
				result.errors = append(result.errors, errMsg)
				mu.Unlock()
				return
			}
			defer conn.Close()

			mu.Lock()
			result.connected++
			mu.Unlock()

			if isHost {
				// wait for all players to connect before starting
				time.Sleep(800 * time.Millisecond)
				if err := conn.WriteJSON(ReqMsg{MsgType: "StartGame", Data: ""}); err != nil {
					fmt.Printf("  ❌ [Room %02d] Host failed to send StartGame: %v\n", roomIndex, err)
				}
			}

			done := make(chan struct{})
			hasGuessed := false // track if already guessed this turn

			go func() {
				defer close(done)
				for {
					_, b, err := conn.ReadMessage()
					if err != nil {
						return
					}
					var resp RespMsg
					if err := json.Unmarshal(b, &resp); err != nil {
						continue
					}

					switch resp.MsgType {
					case "game_started":
						if playerIndex == 0 {
							fmt.Printf("  🎮 [Room %02d] Game started\n", roomIndex)
						}

					case "your_turn":
						// this player is the drawer, reset guess flag
						hasGuessed = false

					case "word_hint":
						// new turn started, reset guess flag and send one guess
						hasGuessed = false
						if !isHost {
							time.Sleep(time.Duration(playerIndex*50) * time.Millisecond)
							if !hasGuessed {
								hasGuessed = true
								if err := conn.WriteJSON(ReqMsg{MsgType: "Guess", Data: "banana"}); err != nil {
									fmt.Printf("  ❌ [Room %02d] Player %d guess failed: %v\n", roomIndex, playerIndex, err)
								}
							}
						}

					case "turn_ended":
						// reset for next turn
						hasGuessed = false
						fmt.Printf("  🔄 [Room %02d] Turn ended\n", roomIndex)

					case "correct_guess":
						var cg struct{ PlayerID string }
						json.Unmarshal(resp.Data, &cg)
						fmt.Printf("  ✅ [Room %02d] Correct guess by %s\n", roomIndex, cg.PlayerID)

					case "game_ended":
						fmt.Printf("  🏁 [Room %02d] Game ended\n", roomIndex)
						mu.Lock()
						gameEndedCount++
						mu.Unlock()
						return
					}
				}
			}()

			select {
			case <-done:
			case <-time.After(300 * time.Second):
				fmt.Printf("  ⚠️  [Room %02d] Player %d timed out\n", roomIndex, playerIndex)
			}
		}(i, token, isHost)

		// small stagger between connections so server isn't slammed
		time.Sleep(20 * time.Millisecond)
	}

	playerWg.Wait()
	result.gameEnded = gameEndedCount > 0

	if result.gameEnded {
		fmt.Printf("  ✅ [Room %02d] PASSED — %d/%d connected, game completed\n", roomIndex, result.connected, numPlayers)
	} else {
		fmt.Printf("  ❌ [Room %02d] FAILED — %d/%d connected, %d errors, game did not complete\n",
			roomIndex, result.connected, numPlayers, result.failed)
	}

	results <- result
}

func main() {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Skribbl Load Test: %d rooms × %d players = %d connections\n",
		numRooms, numPlayers, numRooms*numPlayers)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	var wg sync.WaitGroup
	results := make(chan RoomResult, numRooms)
	start := time.Now()

	for i := 0; i < numRooms; i++ {
		wg.Add(1)
		go runRoom(i, &wg, results)
		time.Sleep(20 * time.Millisecond)
	}

	wg.Wait()
	close(results)
	elapsed := time.Since(start)

	totalConnected := 0
	totalFailed := 0
	gamesCompleted := 0
	var allErrors []string

	for r := range results {
		totalConnected += r.connected
		totalFailed += r.failed
		if r.gameEnded {
			gamesCompleted++
		}
		allErrors = append(allErrors, r.errors...)
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Final Results (%v)\n", elapsed.Round(time.Millisecond))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Rooms launched:     %d\n", numRooms)
	fmt.Printf("  Total connections:  %d / %d\n", totalConnected, numRooms*numPlayers)
	fmt.Printf("  Failed connections: %d\n", totalFailed)
	fmt.Printf("  Rooms completed:    %d / %d\n", gamesCompleted, numRooms)
	if len(allErrors) > 0 {
		fmt.Println("\n  Errors:")
		for _, e := range allErrors {
			fmt.Printf("    - %s\n", e)
		}
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
