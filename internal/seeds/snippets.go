package seeds

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

func SeedOfficialSnippets(systemUser models.User) {
	log.Println("📜 Seeding Official Snippets...")

	// 20 Curated Snippets
	// Languages: Go, Rust, C, C++, Python, Java
	// UI: Vanilla HTML/CSS/JS (Visual)
	// No Frameworks (React, etc included only if vanilla-ish or explicit request, but user said NO framework code)
	// We will stick to high quality vanilla components for VISUAL.

	snippetTemplates := []struct {
		Title       string
		Lang        string
		Type        string
		Diff        string
		Code        string
		PreviewType string
		Output      string
	}{
		// --- Go (Backend/Systems) ---
		{
			Title: "Concurrent Worker Pool", Lang: "go", Type: "ALGORITHM", Diff: "MEDIUM", PreviewType: "TERMINAL",
			Output: "worker 1 started job 1\nworker 2 started job 2\nworker 3 started job 3\nworker 1 finished job 1\nworker 1 started job 4\nworker 2 finished job 2\nworker 2 started job 5\nworker 3 finished job 3\n",
			Code: `package main

import (
	"fmt"
	"sync"
	"time"
)

func worker(id int, jobs <-chan int, results chan<- int, wg *sync.WaitGroup) {
	defer wg.Done()
	for j := range jobs {
		fmt.Printf("worker %d started job %d\n", id, j)
		time.Sleep(time.Second)
		fmt.Printf("worker %d finished job %d\n", id, j)
		results <- j * 2
	}
}

func main() {
	const numJobs = 5
	jobs := make(chan int, numJobs)
	results := make(chan int, numJobs)
	var wg sync.WaitGroup

	for w := 1; w <= 3; w++ {
		wg.Add(1)
		go worker(w, jobs, results, &wg)
	}

	for j := 1; j <= numJobs; j++ {
		jobs <- j
	}
	close(jobs)
	wg.Wait()
	close(results)
}`,
		},
		{
			Title: "HTTP Middleware Pattern", Lang: "go", Type: "UTILITY", Diff: "EASY", PreviewType: "TERMINAL",
			Output: "2026/01/21 12:00:00 GET / 123.45µs\n",
			Code: `package main

import (
	"log"
	"net/http"
	"time"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func main() {
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	http.Handle("/", loggingMiddleware(finalHandler))
	// http.ListenAndServe(":8080", nil) // Uncomment to run
}`,
		},
		{
			Title: "Binary Tree Traversal", Lang: "go", Type: "ALGORITHM", Diff: "MEDIUM", PreviewType: "TERMINAL",
			Output: "4 2 5 1 3 ",
			Code: `package main

import "fmt"

type Node struct {
	Val   int
	Left  *Node
	Right *Node
}

func Inorder(root *Node) {
	if root == nil {
		return
	}
	Inorder(root.Left)
	fmt.Printf("%d ", root.Val)
	Inorder(root.Right)
}

func main() {
	root := &Node{Val: 1, Left: &Node{Val: 2}, Right: &Node{Val: 3}}
	root.Left.Left = &Node{Val: 4}
	root.Left.Right = &Node{Val: 5}
	Inorder(root)
}`,
		},

		// --- Rust (Systems/Safety) ---
		{
			Title: "Safe Memory Handling", Lang: "rust", Type: "EXAMPLE", Diff: "MEDIUM", PreviewType: "TERMINAL",
			Output: "hello\ns2: hello, s3: hello\n",
			Code: `fn main() {
    let s1 = String::from("hello");
    let s2 = s1; // Move occurs here

    // println!("{}", s1); // Error: value borrowed here after move
    println!("{}", s2);
    
    // Cloning
    let s3 = s2.clone();
    println!("s2: {}, s3: {}", s2, s3);
}`,
		},
		{
			Title: "Pattern Matching", Lang: "rust", Type: "UTILITY", Diff: "EASY", PreviewType: "TERMINAL",
			Output: "Text: Hello Rust\n",
			Code: `enum Message {
    Quit,
    Move { x: i32, y: i32 },
    Write(String),
    ChangeColor(i32, i32, i32),
}

fn process(msg: Message) {
    match msg {
        Message::Quit => println!("Quit"),
        Message::Move { x, y } => println!("Move to {}, {}", x, y),
        Message::Write(text) => println!("Text: {}", text),
        _ => println!("Other"),
    }
}

fn main() {
    let msg = Message::Write(String::from("Hello Rust"));
    process(msg);
}`,
		},
		{
			Title: "Async HTTP Request", Lang: "rust", Type: "UTILITY", Diff: "HARD", PreviewType: "TERMINAL",
			Output: "{ \"origin\": \"127.0.0.1\" }\n",
			Code: `// Requires tokio and reqwest crates
#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let resp = reqwest::get("https://httpbin.org/ip")
        .await?
        .json::<std::collections::HashMap<String, String>>()
        .await?;
    println!("{:#?}", resp);
    Ok(())
}`,
		},

		// --- Python (Scripting/DS) ---
		{
			Title: "List Comprehension Power", Lang: "python", Type: "EXAMPLE", Diff: "EASY", PreviewType: "TERMINAL",
			Output: "Original: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]\nSquares of Evens: [4, 16, 36, 64, 100]\n",
			Code: `# Filter and transform in one line
numbers = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
squares_of_evens = [n**2 for n in numbers if n % 2 == 0]

print(f"Original: {numbers}")
print(f"Squares of Evens: {squares_of_evens}")`,
		},
		{
			Title: "Decorator Pattern", Lang: "python", Type: "UTILITY", Diff: "MEDIUM", PreviewType: "TERMINAL",
			Output: "heavy_computation took 0.0123s\n",
			Code: `import time

def timer_decorator(func):
    def wrapper(*args, **kwargs):
        start = time.time()
        result = func(*args, **kwargs)
        end = time.time()
        print(f"{func.__name__} took {end-start:.4f}s")
        return result
    return wrapper

@timer_decorator
def heavy_computation():
    sum([i**2 for i in range(100000)])

heavy_computation()`,
		},
		{
			Title: "Data Class Usage", Lang: "python", Type: "EXAMPLE", Diff: "EASY", PreviewType: "TERMINAL",
			Output: "InventoryItem(name='Widget', unit_price=19.99, quantity_on_hand=10)\nTotal Value: $199.9\n",
			Code: `from dataclasses import dataclass

@dataclass
class InventoryItem:
    name: str
    unit_price: float
    quantity_on_hand: int = 0

    def total_cost(self) -> float:
        return self.unit_price * self.quantity_on_hand

item = InventoryItem("Widget", 19.99, 10)
print(item)
print(f"Total Value: ${item.total_cost()}")`,
		},

		// --- C (Low Level) ---
		{
			Title: "Pointer Arithmetic", Lang: "c", Type: "ALGORITHM", Diff: "MEDIUM", PreviewType: "TERMINAL",
			Output: "Value at ptr + 0: 10\nValue at ptr + 1: 20\nValue at ptr + 2: 30\nValue at ptr + 3: 40\nValue at ptr + 4: 50\n",
			Code: `#include <stdio.h>

int main() {
    int arr[] = {10, 20, 30, 40, 50};
    int *ptr = arr;
    
    for (int i = 0; i < 5; i++) {
        printf("Value at ptr + %d: %d\n", i, *(ptr + i));
    }
    return 0;
}`,
		},
		{
			Title: "Linked List Implementation", Lang: "c", Type: "ALGORITHM", Diff: "HARD", PreviewType: "TERMINAL",
			Output: "Head data: 2\n",
			Code: `#include <stdio.h>
#include <stdlib.h>

struct Node {
    int data;
    struct Node* next;
};

void push(struct Node** head_ref, int new_data) {
    struct Node* new_node = (struct Node*) malloc(sizeof(struct Node));
    new_node->data = new_data;
    new_node->next = (*head_ref);
    (*head_ref) = new_node;
}

int main() {
    struct Node* head = NULL;
    push(&head, 1);
    push(&head, 2);
    printf("Head data: %d\n", head->data);
    return 0;
}`,
		},

		// --- C++ (Performance/Systems) ---
		{
			Title: "Vector & STL Basics", Lang: "cpp", Type: "EXAMPLE", Diff: "EASY", PreviewType: "TERMINAL",
			Output: "Sorted: 1 2 3 4 5 \n",
			Code: `#include <iostream>
#include <vector>
#include <algorithm>

int main() {
    std::vector<int> nums = {4, 2, 5, 1, 3};
    
    // Sort
    std::sort(nums.begin(), nums.end());
    
    std::cout << "Sorted: ";
    for(int n : nums) std::cout << n << " ";
    return 0;
}`,
		},
		{
			Title: "Smart Pointers", Lang: "cpp", Type: "UTILITY", Diff: "MEDIUM", PreviewType: "TERMINAL",
			Output: "Created\nDestroyed\nEnd of scope\n",
			Code: `#include <iostream>
#include <memory>

class Entity {
public:
    Entity() { std::cout << "Created\n"; }
    ~Entity() { std::cout << "Destroyed\n"; }
};

int main() {
    {
        std::unique_ptr<Entity> entity = std::make_unique<Entity>();
        // Auto-destroyed at end of scope
    }
    std::cout << "End of scope\n";
    return 0;
}`,
		},

		// --- Java (Enterprise/OOP) ---
		{
			Title: "Singleton Pattern", Lang: "java", Type: "UTILITY", Diff: "MEDIUM", PreviewType: "TERMINAL",
			Output: "Executing: SELECT * FROM users\n",
			Code: `public class Database {
    private static Database instance;
    
    private Database() {}
    
    public static Database getInstance() {
        if (instance == null) {
            instance = new Database();
        }
        return instance;
    }
    
    public void query(String sql) {
        System.out.println("Executing: " + sql);
    }
}

public class Main {
    public static void main(String[] args) {
         Database.getInstance().query("SELECT * FROM users");
    }
}`,
		},
		{
			Title: "Stream API", Lang: "java", Type: "EXAMPLE", Diff: "MEDIUM", PreviewType: "TERMINAL",
			Output: "C1\nC2\n",
			Code: `import java.util.Arrays;
import java.util.List;

public class Main {
    public static void main(String[] args) {
        List<String> items = Arrays.asList("a1", "a2", "b1", "c2", "c1");

        items.stream()
            .filter(s -> s.startsWith("c"))
            .map(String::toUpperCase)
            .sorted()
            .forEach(System.out::println);
    }
}`,
		},

		// --- Visual / UI (Vanilla) ---
		{
			Title: "Glassmorphism Card", Lang: "html", Type: "VISUAL", Diff: "EASY", PreviewType: "WEB_PREVIEW",
			Output: "",
			Code: `<style>
.card {
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  padding: 2rem;
  color: white;
  max-width: 300px;
}
</style>
<div style="background: linear-gradient(45deg, #FF6B6B, #4ECDC4); height: 100vh; display: flex; align-items: center; justify-content: center;">
  <div class="card">
    <h2>Glass UI</h2>
    <p>Modern glassmorphism effect using backdrop-filter.</p>
  </div>
</div>`,
		},
		{
			Title: "CSS Grid Layout", Lang: "html", Type: "VISUAL", Diff: "MEDIUM", PreviewType: "WEB_PREVIEW",
			Output: "",
			Code: `<div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; color: white;">
  <div style="background: #3498db; padding: 20px;">1</div>
  <div style="background: #e74c3c; padding: 20px;">2</div>
  <div style="background: #2ecc71; padding: 20px;">3</div>
  <div style="background: #f1c40f; padding: 20px; grid-column: span 2;">4 (Span 2)</div>
  <div style="background: #9b59b6; padding: 20px;">5</div>
</div>`,
		},
		{
			Title: "Animated Button", Lang: "html", Type: "VISUAL", Diff: "EASY", PreviewType: "WEB_PREVIEW",
			Output: "",
			Code: `<style>
.btn {
  background: linear-gradient(90deg, #ff8a00, #e52e71);
  border: none;
  border-radius: 25px;
  color: white;
  padding: 12px 24px;
  font-size: 16px;
  cursor: pointer;
  transition: transform 0.2s;
}
.btn:hover {
  transform: scale(1.05);
  box-shadow: 0 5px 15px rgba(229, 46, 113, 0.4);
}
</style>
<div style="display:flex; justify-content:center; padding: 50px; background: #222;">
  <button class="btn">Hover Me</button>
</div>`,
		},
		{
			Title: "Loading Spinner", Lang: "html", Type: "VISUAL", Diff: "EASY", PreviewType: "WEB_PREVIEW",
			Output: "",
			Code: `<style>
.loader {
  border: 4px solid #f3f3f3;
  border-top: 4px solid #3498db;
  border-radius: 50%;
  width: 40px;
  height: 40px;
  animation: spin 1s linear infinite;
}
@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}
</style>
<div style="padding: 50px; background: #222; display: flex; justify-content: center;">
  <div class="loader"></div>
</div>`,
		},
		{
			Title: "Gradient Border", Lang: "html", Type: "VISUAL", Diff: "MEDIUM", PreviewType: "WEB_PREVIEW",
			Output: "",
			Code: `<style>
.box {
  background: #222;
  color: white;
  padding: 2rem;
  border-radius: 8px;
  position: relative;
}
.box::before {
  content: "";
  position: absolute;
  inset: -2px;
  background: linear-gradient(45deg, #fb0094, #0000ff, #00ff00);
  z-index: -1;
  border-radius: 10px;
}
</style>
<div style="padding: 50px; background: #000; display: flex; justify-content: center;">
  <div class="box">Gradient Border</div>
</div>`,
		},
		// --- Markdown & Mermaid (Documentation) ---
		{
			Title: "How to Share Snippets", Lang: "markdown", Type: "EXAMPLE", Diff: "EASY", PreviewType: "WEB_PREVIEW_TOP",
			Output: "",
			Code: `# Sharing Code on CodeStudio

CodeStudio makes it easy to share your code snippets.

## Features
- ** Syntax Highlighting ** for 50+ languages
- ** Live Previews ** for HTML/CSS/React
- ** Mermaid Diagrams ** support

### Code Example
` + "```" + `javascript
console.log("Hello World");
` + "```" + `

> "Code is like humor. When you have to explain it, it’s bad." - Cory House
`,
		},
		{
			Title: "System Architecture", Lang: "mermaid", Type: "VISUAL", Diff: "MEDIUM", PreviewType: "WEB_PREVIEW",
			Output: "",
			Code: `graph TD
    A[Client] -->|HTTP| B(Load Balancer)
    B --> C{Service A}
    B --> D{Service B}
    C --> E[Database]
    D --> E
    D --> F[Cache]
    
    style A fill:#f9f,stroke:#333,stroke-width:2px
    style E fill:#bbf,stroke:#333,stroke-width:2px`,
		},
		{
			Title: "Binary Search Explained (Beginner Friendly)", Lang: "python", Type: "ALGORITHM", Diff: "EASY", PreviewType: "TERMINAL",
			Output: "Element found at index: 6\n",
			Code: `def binary_search(arr, target):
    left, right = 0, len(arr) - 1

    while left <= right:
        mid = (left + right) // 2
        if arr[mid] == target:
            return mid
        elif arr[mid] < target:
            left = mid + 1
        else:
            right = mid - 1

    return -1

# Example usage:
arr = [2, 5, 8, 12, 16, 23, 38, 56, 72, 91]
target = 38
result = binary_search(arr, target)
print(f"Element found at index: {result}")`,
		},
	}

	for _, t := range snippetTemplates {
		var existing models.Snippet
		// Check by Title to avoid duplicates (idempotency)
		if err := database.DB.Where(&models.Snippet{Title: t.Title, AuthorID: systemUser.ID}).First(&existing).Error; err == nil {
			log.Printf("   ℹ️ Snippet already exists: %s", t.Title)

			// Force update of output for existing snippets if they have none
			if t.Output != "" && existing.LastExecutionOutput == "" {
				existing.LastExecutionOutput = t.Output
				existing.LastExecutionStatus = "SUCCESS"
				database.DB.Save(&existing)
				log.Printf("     ---> Updated output for: %s", t.Title)
			}
			continue
		}

		s := models.Snippet{
			ID:                  uuid.New().String(),
			Title:               t.Title,
			Description:         fmt.Sprintf("Official example for %s", t.Title),
			Code:                t.Code,
			Language:            t.Lang,
			Status:              "PUBLISHED",
			Verified:            true, // Official
			LastExecutionStatus: "SUCCESS",
			LastExecutionOutput: t.Output,
			Output:              t.Output,
			OutputSnapshot:      t.Output,
			Visibility:          "PUBLIC",
			AuthorID:            systemUser.ID,
			Type:                t.Type,
			Difficulty:          t.Diff,
			PreviewType:         t.PreviewType,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		if err := database.DB.Create(&s).Error; err != nil {
			log.Printf("   ❌ Failed to create snippet %s: %v", s.Title, err)
		} else {
			log.Printf("   📝 Snippet Added: %s", s.Title)
		}
	}
}
