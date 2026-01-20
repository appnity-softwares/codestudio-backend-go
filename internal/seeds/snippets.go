package seeds

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

func SeedSnippets(creator models.User) {
	log.Println("📜 Seeding Code Snippets...")

	snippets := []models.Snippet{
		{
			ID:                  uuid.New().String(),
			Title:               "Depth First Search",
			Description:         "A standard DFS implementation in Python",
			Code:                "def dfs(node, visited):\n    if node in visited:\n        return\n    visited.add(node)\n    print(f'Visiting {node}')\n    for neighbor in [node+1]: # Dummy neighbors\n        if neighbor < 5: dfs(neighbor, visited)\n\nvisited = set()\ndfs(0, visited)",
			Language:            "python",
			Status:              "PUBLISHED",
			Verified:            true,
			LastExecutionStatus: "SUCCESS",
			OutputSnapshot:      `{"run":{"stdout":"Visiting 0\nVisiting 1\nVisiting 2\nVisiting 3\nVisiting 4\n","stderr":"","code":0}}`,
			Visibility:          "PUBLIC",
			AuthorID:            creator.ID,
			PreviewType:         "TERMINAL",
		},
		{
			ID:          uuid.New().String(),
			Title:       "Modern Neumorphic Card",
			Description: "A beautiful UI component using HTML/CSS",
			Code: `<div style="padding: 40px; background: #e0e0e0; border-radius: 50px; box-shadow: 20px 20px 60px #bebebe, -20px -20px 60px #ffffff; color: #444; text-align: center; font-family: sans-serif;">
  <h1 style="margin-bottom: 20px;">Neumorphism</h1>
  <p>Soft UI is the new trend.</p>
  <button style="margin-top: 20px; padding: 12px 24px; border-radius: 12px; background: #e0e0e0; border: none; box-shadow: 5px 5px 10px #bebebe, -5px -5px 10px #ffffff; cursor: pointer; color: #666; font-weight: bold;">Click Me</button>
</div>`,
			Language:            "html",
			Status:              "PUBLISHED",
			Verified:            true,
			LastExecutionStatus: "SUCCESS",
			Visibility:          "PUBLIC",
			AuthorID:            creator.ID,
			PreviewType:         "WEB_PREVIEW",
		},
		{
			ID:          uuid.New().String(),
			Title:       "Glassmorphism Counter",
			Description: "Interactive React component with glassmorphism",
			Code: `import React, { useState } from 'react';

export default function Counter() {
  const [count, setCount] = useState(0);

  return (
    <div style={{
      padding: '40px',
      background: 'rgba(255, 255, 255, 0.1)',
      backdropFilter: 'blur(10px)',
      borderRadius: '24px',
      border: '1px solid rgba(255, 255, 255, 0.2)',
      color: 'white',
      textAlign: 'center',
      fontFamily: 'system-ui'
    }}>
      <h2 style={{ margin: '0 0 20px 0' }}>React Glass Counter</h2>
      <div style={{ fontSize: '48px', fontWeight: 'bold', margin: '20px 0' }}>{count}</div>
      <button 
        onClick={() => setCount(c => c + 1)}
        style={{
          padding: '10px 20px',
          borderRadius: '12px',
          background: '#6366f1',
          color: 'white',
          border: 'none',
          cursor: 'pointer',
          fontWeight: 'bold'
        }}
      >
        Increment
      </button>
    </div>
  );
}`,
			Language:            "react",
			Status:              "PUBLISHED",
			Verified:            true,
			LastExecutionStatus: "SUCCESS",
			Visibility:          "PUBLIC",
			AuthorID:            creator.ID,
			PreviewType:         "WEB_PREVIEW",
		},
		{
			ID:                  uuid.New().String(),
			Title:               "Fibonacci Sequence",
			Description:         "Efficient recursive approach with memoization",
			Code:                "function fib(n, memo = {}) {\n  if (n in memo) return memo[n];\n  if (n <= 2) return 1;\n  memo[n] = fib(n - 1, memo) + fib(n - 2, memo);\n  return memo[n];\n}\n\nconsole.log(fib(10));",
			Language:            "typescript",
			Status:              "PUBLISHED",
			Verified:            true,
			LastExecutionStatus: "SUCCESS",
			OutputSnapshot:      `{"run":{"stdout":"55\n","stderr":"","code":0}}`,
			Visibility:          "PUBLIC",
			AuthorID:            creator.ID,
			PreviewType:         "TERMINAL",
		},
	}

	for _, s := range snippets {
		s.CreatedAt = time.Now()
		s.UpdatedAt = time.Now()
		if err := database.DB.Create(&s).Error; err != nil {
			log.Printf("   ❌ Failed to create snippet %s: %v", s.Title, err)
		} else {
			log.Printf("   📝 Snippet Added: %s (%s)", s.Title, s.Language)
		}
	}
}
