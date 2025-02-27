package main

import (
    "bufio"
    "context"
    "fmt"
    "log"
    "os"
    "strings"

    "github.com/google/generative-ai-go/genai"
    "google.golang.org/api/option"
)

const endOfStreamErr = "no more items in iterator"

// func checkInternet() bool {
//     client := &http.Client{
//         Timeout: 5 * time.Second,
//     }
//     resp, err := client.Get("https://www.google.com")
//     if err != nil {
//         return false
//     }
//     resp.Body.Close()
//     return true
// }

func initializeAPI() (string, string) {
    // if !checkInternet() {
    //     log.Fatal("❌ No internet connection. Please check your network and try again.")
    // }

    configFile := "config.txt"
    var apiKey, model string

    if file, err := os.Open(configFile); err == nil {
        scanner := bufio.NewScanner(file)
        config := make(map[string]string)
        for scanner.Scan() {
            parts := strings.SplitN(scanner.Text(), "=", 2)
            if len(parts) == 2 {
                config[parts[0]] = parts[1]
            }
        }
        file.Close()
        apiKey, model = config["api"], config["model"]
    }

    if apiKey == "" || model == "" {
        fmt.Println("🔑 API key and model required.")
        fmt.Print("Enter API Key: ")
        scanner := bufio.NewScanner(os.Stdin)
        scanner.Scan()
        apiKey = strings.TrimSpace(scanner.Text())

        fmt.Print("Enter Model: ")
        scanner.Scan()
        model = strings.TrimSpace(scanner.Text())

        file, err := os.Create(configFile)
        if err == nil {
            file.WriteString(fmt.Sprintf("api=%s\nmodel=%s\n", apiKey, model))
            file.Close()
        }
    }

    return apiKey, model
}

func chatWithGemini(ctx context.Context, client *genai.Client, query, instruction, model string) {
    gm := client.GenerativeModel(model)
    content := genai.Text(instruction + " " + query)
    iter := gm.GenerateContentStream(ctx, content)
    if iter == nil {
        log.Fatalf("Error initiating stream")
    }

    for {
        resp, err := iter.Next()
        if err != nil {
            if strings.Contains(err.Error(), endOfStreamErr) {
                break
            }
            log.Fatalf("Error streaming message: %v", err)
        }
        for _, part := range resp.Candidates[0].Content.Parts {
            fmt.Printf("%v ", part)
            os.Stdout.Sync()
        }
    }
    fmt.Println()
}

func interactiveMode(ctx context.Context, client *genai.Client, instruction, model string) {
    fmt.Println("💬 Interactive mode enabled (Type 'exit' to quit)")

    gm := client.GenerativeModel(model)
    session := gm.StartChat()
    session.SendMessage(ctx, genai.Text(instruction))

    scanner := bufio.NewScanner(os.Stdin)
    for {
        fmt.Print("You: ")
        scanner.Scan()
        userInput := strings.TrimSpace(scanner.Text())
        if userInput == "exit" || userInput == "quit" {
            fmt.Println("👋 Goodbye!")
            break
        }
        iter := session.SendMessageStream(ctx, genai.Text(userInput))
        if iter == nil {
            log.Fatalf("Error initiating stream")
        }
        fmt.Print("AI: ")
        for {
            resp, err := iter.Next()
            if err != nil {
                if strings.Contains(err.Error(), endOfStreamErr) {
                    break
                }
                log.Fatalf("Error streaming message: %v", err)
            }
            for _, part := range resp.Candidates[0].Content.Parts {
                fmt.Printf("%v ", part)
                os.Stdout.Sync()
            }
        }
        fmt.Println()
    }
}

func main() {
    ctx := context.Background()
    apiKey, model := initializeAPI()

    client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
    if err != nil {
        log.Fatalf("Error creating client: %v", err)
    }
    defer client.Close()

    args := os.Args[1:]
    interactive := false
    instruction := "You are an AI assistant."
    var forceConcise *bool

    if len(args) > 0 && strings.HasPrefix(args[0], "-") {
        flags := args[0][1:]
        queryParts := args[1:]

        interactive = strings.Contains(flags, "i")
        if strings.Contains(flags, "t") {
            instruction = "Act as a Linux/CMD/Powershell command expert and help me fix commands. Only provide the command with a minimal comment, as your output will go directly to the terminal."
        } else {
            if interactive {
                f := strings.Contains(flags, "f")
                forceConcise = &f
            } else {
                n := !strings.Contains(flags, "n")
                forceConcise = &n
            }
        }

        if forceConcise != nil && *forceConcise {
            instruction = "Stay to the point and say less."
        }

        if interactive || len(queryParts) == 0 {
            interactiveMode(ctx, client, instruction, model)
            return
        }

        query := strings.Join(queryParts, " ")
        chatWithGemini(ctx, client, query, instruction, model)
        return
    }

    if len(args) == 0 {
        interactiveMode(ctx, client, instruction, model)
    } else {
        query := strings.Join(args, " ")
        chatWithGemini(ctx, client, query, "Stay to the point and say less.", model)
	   }
}