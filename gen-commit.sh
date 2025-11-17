#!/bin/bash
CHANGES=$(git diff --cached)
COMMIT_MSG=$(curl -s https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "content-type: application/json" \
  -d "{
    \"model\": \"claude-haiku-4-5\",
    \"max_tokens\": 100,
    \"messages\": [{
      \"role\": \"user\",
      \"content\": \"生成簡潔的 git commit message:\n$CHANGES\"
    }]
  }" | jq -r '.content[0].text')

echo "$COMMIT_MSG"
```

然後在 Obsidian 中執行:
```
bash gen-commit.sh