Output demo/gifs/hero.gif
Set Height 820
Source demo/tapes/_setup.tape

Type "# Delegate mode: CommitCraft hands the agent a prompt bundle — no Groq calls." Enter
Sleep 1s
Type "commitcraft ai generate -k 'retry transient Groq 429s with backoff' -t ADD -s api --agent | jq '{mode,strategy,inputs}'" Sleep 400ms Enter
Sleep 2.6s

Type "# The agent writes the message, submits it, and CommitCraft verifies it:" Enter
Sleep 1s
Type "commitcraft ai submit --input-file msg.json | jq '.verify'" Sleep 400ms Enter
Sleep 2.4s

Type "# The finished, ready-to-commit message:" Enter
Sleep 1s
Type "commitcraft ai show --id 1 | jq -r .final_message" Sleep 400ms Enter
Sleep 3.5s
