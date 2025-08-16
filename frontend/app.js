document.addEventListener('DOMContentLoaded', () => {
    // --- DOM Elements ---
    const homeView = document.getElementById('home-view');
    const boardView = document.getElementById('board-view');
    const createBoardBtn = document.getElementById('create-board-btn');
    const huddleLinkInput = document.getElementById('huddle-link-input');
    const editorContainer = document.getElementById('editor-container');
    const taskList = document.getElementById('task-list');
    const taskInput = document.getElementById('task-input');
    const addTaskBtn = document.getElementById('add-task-btn');
    const exportBtn = document.getElementById('export-btn');

    // --- State Variables ---
    let ws;
    let editor;
    let tasks = [];
    let programmaticChange = false;
    let huddleLink = "";
    const debounceTimers = {};

    // --- Debounce Utility ---
    function debounce(key, func, delay) {
        clearTimeout(debounceTimers[key]);
        debounceTimers[key] = setTimeout(func, delay);
    }

    // --- Monaco Editor Initialization ---
    require.config({ paths: { 'vs': 'https://cdn.jsdelivr.net/npm/monaco-editor@0.44.0/min/vs' }});
    require(['vs/editor/editor.main'], () => {
        editor = monaco.editor.create(editorContainer, {
            value: "// Loading content...",
            language: 'javascript',
            theme: 'vs-dark',
            automaticLayout: true,
        });
        
        // When editor content changes, send update after a debounce period
        editor.getModel().onDidChangeContent(() => {
            if (programmaticChange) return;
            debounce('code', () => {
                sendMessage({ type: "CODE_UPDATE", payload: editor.getValue() });
            }, 500); // Send update 500ms after user stops typing
        });
    });

    // --- WebSocket Logic ---
    function connectWebSocket(boardId) {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        ws = new WebSocket(`${protocol}//${window.location.host}/ws/${boardId}`);
        ws.onmessage = handleWebSocketMessage;
    }

    function handleWebSocketMessage(event) {
        const msg = JSON.parse(event.data);
        programmaticChange = true;

        switch (msg.type) {
            case 'INITIAL_STATE':
                updateFullState(msg.payload);
                break;
            case 'CODE_UPDATE':
                if (editor && editor.getValue() !== msg.payload) {
                    editor.setValue(msg.payload);
                }
                break;
            case 'TASKS_UPDATE':
                tasks = msg.payload || [];
                renderTasks();
                break;
            case 'LINK_UPDATE':
                 if (huddleLinkInput.value !== msg.payload) {
                    huddleLinkInput.value = msg.payload;
                }
                break;
        }
        
        setTimeout(() => programmaticChange = false, 50);
    }
    
    function updateFullState(state) {
        if (editor) {
            editor.setValue(state.contentCode);
            // Auto-detect language based on common patterns
            const model = editor.getModel();
            if (state.contentCode.trim().startsWith('{')) {
                monaco.editor.setModelLanguage(model, 'json');
            } else if (state.contentCode.includes('package main')) {
                 monaco.editor.setModelLanguage(model, 'go');
            }
        }
        huddleLinkInput.value = state.huddleLink || "";
        tasks = state.contentTasks || [];
        renderTasks();
    }
    
    function sendMessage(data) {
        if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify(data));
    }

    // --- UI Update & Task Logic ---
    function renderTasks() {
        taskList.innerHTML = '';
        tasks.forEach(task => {
            const li = document.createElement('li');
            li.className = task.completed ? 'completed' : '';
            li.innerHTML = `
                <input type="checkbox" data-id="${task.id}" ${task.completed ? 'checked' : ''}>
                <span>${escapeHTML(task.text)}</span>
                <button class="delete-task" data-id="${task.id}">&times;</button>
            `;
            taskList.appendChild(li);
        });
    }

    function handleTaskUpdate() {
        sendMessage({ type: "TASKS_UPDATE", payload: tasks });
    }

    // --- Event Listeners & Routing ---
    createBoardBtn.addEventListener('click', () => {
        const boardId = Math.random().toString(36).substring(2, 12);
        window.location.href = `/b/${boardId}`;
    });

    huddleLinkInput.addEventListener('input', () => {
        debounce('link', () => {
            sendMessage({ type: "LINK_UPDATE", payload: huddleLinkInput.value });
        }, 500);
    });
    
    addTaskBtn.addEventListener('click', () => {
        const text = taskInput.value.trim();
        if (text) {
            tasks.push({ id: `task-${Date.now()}`, text, completed: false });
            taskInput.value = '';
            renderTasks();
            handleTaskUpdate();
        }
    });
    taskInput.addEventListener('keydown', e => { if (e.key === 'Enter') addTaskBtn.click(); });
    
    taskList.addEventListener('click', e => {
        const target = e.target;
        if (target.type === 'checkbox') {
            const task = tasks.find(t => t.id === target.dataset.id);
            if (task) {
                task.completed = target.checked;
                renderTasks();
                handleTaskUpdate();
            }
        }
        if (target.classList.contains('delete-task')) {
            tasks = tasks.filter(t => t.id !== target.dataset.id);
            renderTasks();
            handleTaskUpdate();
        }
    });
    
    exportBtn.addEventListener('click', () => {
        const markdown = `# Huddle Export\n\n## Meeting Link\n${huddleLinkInput.value || 'Not set'}\n\n## Tasks\n${tasks.map(t => `- [${t.completed ? 'x' : ' '}] ${t.text}`).join('\n')}\n\n## Code\n\`\`\`\n${editor.getValue()}\n\`\`\``;
        const blob = new Blob([markdown], { type: 'text/markdown' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `huddle-export-${new Date().toISOString()}.md`;
        a.click();
        URL.revokeObjectURL(url);
    });

    function escapeHTML(str) { return str.replace(/</g, "&lt;").replace(/>/g, "&gt;"); }

    // --- Initial Page Load & Routing ---
    function route() {
        const path = window.location.pathname;
        const match = path.match(/^\/b\/([a-zA-Z0-9-]+)/);
        if (match) {
            const boardId = match[1];
            homeView.classList.remove('active');
            boardView.classList.add('active');
            connectWebSocket(boardId);
        } else {
            homeView.classList.add('active');
            boardView.classList.remove('active');
        }
    }
    
    route();
});