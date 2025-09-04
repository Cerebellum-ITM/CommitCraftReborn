# CommitCraft: AI-Powered Commit Assistant for Developers

CommitCraft is a powerful Terminal User Interface (TUI) tool designed to streamline your Git workflow. It leverages Artificial Intelligence (AI) to generate meaningful, concise, and well-formatted commit messages based on your staged changes. With an interactive experience powered by [Charmbracelet Bubble Tea](https://github.com/charmbracelet/bubbletea), CommitCraft helps you create high-quality commits quickly and effortlessly.

## ✨ Key Features

- **AI-Powered Commit Message Generation:** Utilize Groq AI to suggest commit messages based on your staged changes.
- **Interactive Terminal User Interface (TUI):** A smooth and visually appealing user experience built with the Charmbracelet suite (Bubble Tea, Lipgloss, Glamour).
- **Customizable Commit Types:** Define and manage your own commit types (e.g., `feat`, `fix`, `docs`, `style`) with descriptions and colors.
- **Commit Scope Selection:** Select specific files or directories to define the scope of your commit.
- **Flexible Commit Format:** Customize how your final commit message is structured.
- **Customizable AI Prompts:** Adjust the prompt templates used by the AI for full control over the suggestions.
- **Nerd Fonts Support:** Enhance the TUI aesthetics with Nerd Fonts icons for better file and directory visualization.
- **Commit History:** Ability to view and manage previous commits (WIP).

## 🚀 Installation

CommitCraft is written in Go, making it easy to compile and install across different platforms.

### Requirements

- [Git](https://git-scm.com/) (required for repository interaction)
- [Go 1.21+](https://go.dev/doc/install) (for compiling from source)

### From Source

1. **Clone the repository:**

    ```bash
    git clone https://github.com/your-username/CommitCraft_v2.git
    cd CommitCraft_v2
    ```

    (Replace `https://github.com/your-username/CommitCraft_v2.git` with your actual repository URL)

2. **Compile the binary:**
    You can compile the binary for your current operating system:

    ```bash
    go build -o commitcraft ./cmd/cli
    ```

    To cross-compile for other platforms, you can use `GOOS` and `GOARCH`:

    - **Linux (64-bit Intel/AMD):**

        ```bash
        GOOS=linux GOARCH=amd64 go build -o commitcraft_linux_amd64 ./cmd/cli
        ```

    - **macOS (64-bit Intel):**

        ```bash
        GOOS=darwin GOARCH=amd64 go build -o commitcraft_darwin_amd64 ./cmd/cli
        ```

    - **macOS (64-bit Apple Silicon):**

        ```bash
        GOOS=darwin GOARCH=arm64 go build -o commitcraft_darwin_arm64 ./cmd/cli
        ```

    - **Windows (64-bit):**

        ```bash
        GOOS=windows GOARCH=amd64 go build -o commitcraft_windows_amd64.exe ./cmd/cli
        ```

3. **Move the binary to your PATH:**
    After compiling, move the resulting binary to a directory that is in your `PATH` (e.g., `/usr/local/bin` or `~/.local/bin`):

    ```bash
    sudo mv commitcraft /usr/local/bin/ # For global installation
    # Or for your user:
    mkdir -p ~/.local/bin
    mv commitcraft ~/.local/bin/
    ```

### From a GitHub Release (Recommended)

Once releases are available, the easiest way is to download the appropriate pre-compiled binary for your system from the [CommitCraft Releases page](https://github.com/your-username/CommitCraft_v2/releases).

1. Download the `commitcraft_<OS>_<ARCH>` file (or `.exe` for Windows).
2. Unzip it if necessary.
3. Move the unzipped binary to a directory in your `PATH` (e.g., `/usr/local/bin` or `~/.local/bin`).
4. Ensure the binary has execute permissions: `chmod +x /path/to/commitcraft`

## ⚙️ Configuration

CommitCraft uses `TOML` configuration files for flexible customization.

### Configuration File Locations

- **Global Configuration:** `~/.config/commitcraft/config.toml`
  - If this file does not exist, it will be created automatically with default settings the first time CommitCraft runs.
- **Local Configuration:** `.commitcraft.toml` in the root directory of your current Git repository.
  - Local configuration **overrides** global configuration.

### Groq API Key

CommitCraft requires an API Key from [Groq](https://groq.com/) to interact with its AI models.
You can set it up in two ways:

1. **Environment Variable (Recommended & Secure):**
    Set the `GROQ_API_KEY` environment variable in your shell (e.g., `~/.bashrc`, `~/.zshrc`, or `~/.profile`):

    ```bash
    export GROQ_API_KEY="your_groq_api_key_here"
    ```

    Make sure to **restart your terminal** or run `source ~/.bashrc` (or similar) for changes to take effect.

2. **Interactive Setup:**
    If `GROQ_API_KEY` is not set, CommitCraft will prompt you for the API Key the first time it runs and save it to the global configuration.

### Customizing Commit Types

You can define your own commit types in your configuration file (`config.toml` or `.commitcraft.toml`).

Example `.commitcraft.toml` for adding custom commit types:

```toml
# .commitcraft.toml
[commit_types]
behavior = "append" # Or "replace" to use only your custom types

[[commit_types.types]]
tag = "STYLE"
description = "Formatting and style adjustments that do not change the meaning of the code."
color = "#E57373" # You can use Hex color codes (#RRGGBB) or Lipgloss color names

[[commit_types.types]]
tag = "TEST"
description = "Adding or correcting tests (unit, integration, e2e)."
color = "#81D4FA"

[[commit_types.types]]
tag = "PERF"
description = "Performance improvements."
color = "#FFB74D"
```

### Customizing AI Prompts

The prompts used by the AI to generate suggestions are templates that you can modify. These files are located in:
`~/.config/commitcraft/prompts/`

The files are:

- `summary.prompt.tmpl`: For summarizing changes.
- `commit_builder.prompt.tmpl`: For building the final commit message.
- `output_format.prompt.tmpl`: For formatting the final commit output.

You can edit these files to tailor the AI's behavior to your needs.

### Nerd Fonts Usage

If you have [Nerd Fonts](https://www.nerdfonts.com/) installed on your system and terminal, you can enable their use in the TUI for better file icon visualization.
In `~/.config/commitcraft/config.toml` (or `.commitcraft.toml`):

```toml
[tui]
use_nerd_fonts = true # Set to false to disable
```

## 🚀 Usage

Once installed and configured, you can run CommitCraft from your terminal:

```bash
commitcraft
```

**Tip:** For quicker access, consider adding an alias to your shell (e.g., `~/.bashrc` or `~/.zshrc`):

```bash
alias gc='commitcraft'
```

Then, simply run `gc` in your terminal.

### Basic Workflow

1. **Stage your changes:** Use `git add <files>` or `git add .` as you normally would.
2. **Run CommitCraft:** `commitcraft` (or `gc`).
3. **Follow the TUI:**
    - Select the commit type.
    - Select the scope (files or directories).
    - Write your base message, and the AI will offer suggestions or translations.
    - Confirm the final message.
4. **Commit Created:** CommitCraft will execute `git commit` with the generated message.

## 🤝 Contributing

Contributions are welcome! If you're interested in improving CommitCraft, please open an issue or submit a Pull Request.

## 📄 License

This project is under the [MIT License](LICENSE). <!-- Make sure to have a LICENSE file -->

---
