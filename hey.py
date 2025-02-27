import sys
import os
import google.generativeai as genai
from google.api_core import exceptions
import requests

os.environ["GRPC_ENABLE_FORK_SUPPORT"] = "0"
os.environ["GRPC_VERBOSITY"] = "NONE"
os.environ["GRPC_POLL_STRATEGY"] = "poll"

def check_internet():
    try:
        requests.get("https://www.google.com", timeout=5)
        return True
    except requests.ConnectionError:
        return False

def exception_handler(func):
    def wrapper(*args, **kwargs):
        try:
            return func(*args, **kwargs)
        except exceptions.InvalidArgument:
            print("❌ Invalid API Key or Argument. Deleting config file.")
            if os.path.exists("config.txt"):
                os.remove("config.txt")
            sys.exit(1)
        except exceptions.NotFound:
            print("❌ Invalid model name. Deleting config file.")
            if os.path.exists("config.txt"):
                os.remove("config.txt")
            sys.exit(1)
        except exceptions.ResourceExhausted:
            print("⏳ Rate limit exceeded. Try again later.")
            sys.exit(1)
        except exceptions.GoogleAPIError as e:
            print(f"❌ API error: {e}")
            sys.exit(1)
        except Exception as e:
            print(f"⚠️ Unexpected error: {e}")
            sys.exit(1)
    return wrapper

def initialize_api():
    if not check_internet():
        print("❌ No internet connection. Please check your network and try again.")
        sys.exit(1)
    
    config_file = "config.txt"
    api_key, model = None, None
    
    if os.path.exists(config_file):
        try:
            with open(config_file, "r") as file:
                lines = file.read().splitlines()
                config = dict(line.split("=", 1) for line in lines if "=" in line)
                api_key = config.get("api")
                model = config.get("model")
        except Exception:
            print("⚠️ Error reading config file. Re-enter details.")
    
    if not api_key or not model:
        print("🔑 API key and model required.")
        api_key = input("Enter API Key: ").strip()
        model = input("Enter Model: ").strip()
        with open(config_file, "w") as file:
            file.write(f"api={api_key}\nmodel={model}")
    
    try:
        genai.configure(api_key=api_key)
        return model
    except Exception as e:
        print(f"❌ Failed to initialize API: {e}")
        sys.exit(1)

@exception_handler
def chat_with_gemini(query, instruction, model):
    model_instance = genai.GenerativeModel(model)
    history = model_instance.start_chat()
    history.send_message(instruction)
    
    response = history.send_message(query, stream=True)
    for chunk in response:
        words = chunk.text.split()
        for word in words:
            print(word, end=" ", flush=True)
    print("\n")

@exception_handler
def interactive_mode(instruction, model):
    os.system("cls" if os.name == "nt" else "clear")
    print("💬 Interactive mode enabled (Type 'exit' to quit)\n")
    
    model_instance = genai.GenerativeModel(model)
    history = model_instance.start_chat()
    history.send_message(instruction)
    
    while True:
        user_input = input("You: ").strip()
        if user_input.lower() in ["exit", "quit"]:
            print("👋 Goodbye!")
            break
        response = history.send_message(user_input, stream=True)
        print("AI:", end=" ", flush=True)
        for chunk in response:
            words = chunk.text.split()
            for word in words:
                print(word, end=" ", flush=True)
        print("\n")

def main():
    model = initialize_api()
    args = sys.argv[1:]
    
    interactive = False
    instruction = "You are an AI assistant."
    force_concise = None
    
    if args and args[0].startswith("-"):
        flags = args[0][1:]
        query_parts = args[1:]

        interactive = "i" in flags
        if "t" in flags:
            instruction = "Act as a Linux/CMD/Powershell command expert and help me fix commands. Only provide the command with a minimal comment, as your output will go directly to the terminal."
        else:
            if interactive:
                force_concise = "f" in flags
            else:
                force_concise = "n" not in flags
        
        if force_concise:
            instruction = f"Stay to the point and say less."
    else:
        query_parts = args

    if interactive or not query_parts:
        interactive_mode(instruction, model)
        return
    
    if instruction == "You are an AI assistant.":
        instruction = f"Stay to the point and say less."

    query = " ".join(query_parts)
    chat_with_gemini(query, instruction, model)

if __name__ == "__main__":
    main()
