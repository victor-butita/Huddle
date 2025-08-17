# Huddle 🚀  
The instant, ephemeral command center for your team.  

Huddle is a real-time collaborative workspace designed for quick, focused team sessions.  
Create an instant, shareable board with a collaborative code editor, task manager, notes, and integrated voice chat.  
No sign-ups, no saved history — just pure, in-the-moment collaboration.  

---

## ✨ Features

- ⚡ **Real-time Code Editor**: A shared Monaco Editor instance lets your team code together live, with support for dozens of languages.  
- 🤖 **AI-Powered Code Assistant**:  
  - **Analyze**: Get instant explanations and complexity analysis of code.  
  - **Refactor**: Request performance and readability improvements.  
  - **Add Comments**: Automatically generate documentation for your functions.  
- ✅ **Collaborative Task Board**: Create, assign, and track tasks with a live-updating progress bar.  
- 🗣️ **Integrated Voice Chat**: Jump into a voice huddle directly in the app, powered by the Jitsi Meet API.  
- 🔒 **Ephemeral Rooms**: Create a board with a single click. Share the URL to invite teammates instantly.  
- ✍️ **Shared Notes & Links**: A scratchpad for notes plus a field for a shared video call link.  
- 💬 **Live Chat**: Simple, real-time chat for quick messages and links.  

---

## 🛠 Tech Stack  

**Backend**  
- Go  
- Routing: [Gorilla Mux](https://github.com/gorilla/mux)  
- Real-time Communication: [Gorilla WebSocket](https://github.com/gorilla/websocket)  
- Persistence: [SQLite](https://github.com/mattn/go-sqlite3)  

**Frontend**  
- HTML5  
- CSS3  
- JavaScript (ES6+)  

**Key Libraries**  
- [Monaco Editor](https://microsoft.github.io/monaco-editor/): The code editor that powers VS Code.  
- [Jitsi Meet API](https://jitsi.github.io/handbook/docs/dev-guide/dev-guide-iframe): For integrated voice chat.  
- [Marked.js](https://marked.js.org/): For rendering AI responses in Markdown.  

**APIs**  
- OpenAI API (or compatible) for AI code assistance.  

---

## 🚀 Getting Started  

Follow these instructions to get a local copy up and running.  

### ✅ Prerequisites  
- **Go**: Version 1.18 or higher. [Install Go](https://go.dev/dl/)  
- **OpenAI API Key**: Required for AI features. [Get an API Key](https://platform.openai.com/)  

### 📥 Installation & Setup  

Clone the repository:  
```sh
git clone https://github.com/your-username/huddle.git
cd huddle
