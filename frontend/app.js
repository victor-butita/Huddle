document.addEventListener('DOMContentLoaded', () => {
    // --- State Management ---
    const state = {
        boardId: null, ws: null, editor: null, jitsiApi: null, editorReadyPromise: null,
        user: { id: null, name: "Guest", color: getRandomColor() },
        team: [], tasks: [], programmaticChange: false, debounceTimers: {}
    };

    // --- DOM Element Cache ---
    const DOM = {
        homeView: document.getElementById('home-view'), boardView: document.getElementById('board-view'),
        createBoardBtn: document.getElementById('create-board-btn'), navTabs: document.querySelectorAll('.nav-tab'),
        mainPanels: document.querySelectorAll('.main-panel'), editorContainer: document.getElementById('editor-container'),
        languageSelect: document.getElementById('language-select'), aiToolbarButtons: document.querySelectorAll('.toolbar-btn'),
        taskBoard: document.getElementById('task-board'), progressBar: document.getElementById('progress-bar-inner'),
        progressText: document.getElementById('progress-text'), notesEditor: document.getElementById('notes-editor'),
        teamList: document.getElementById('team-list'), huddleLinkInput: document.getElementById('huddle-link-input'),
        chatMessages: document.getElementById('chat-messages'), chatInput: document.getElementById('chat-input'),
        sendChatBtn: document.getElementById('send-chat-btn'), modalOverlay: document.getElementById('modal-overlay'),
        onboardingModal: document.getElementById('onboarding-modal'), aiModal: document.getElementById('ai-modal'),
        userAvatarSetup: document.getElementById('user-avatar-setup'), userNameSetup: document.getElementById('user-name-setup'),
        joinHuddleBtn: document.getElementById('join-huddle-btn'), aiModalTitle: document.getElementById('ai-modal-title'),
        aiModalText: document.getElementById('ai-modal-text'), aiModalCloseBtn: document.getElementById('ai-modal-close-btn'),
        editUserBtn: document.getElementById('edit-user-btn'), voiceChatContainer: document.getElementById('voice-chat-container'),
        joinVoiceBtn: document.getElementById('join-voice-btn')
    };

    // --- Core Application Logic ---
    function init() {
        console.log("Huddle App Initializing...");
        setupEventListeners();
        route();
    }

    function route() {
        const match = window.location.pathname.match(/^\/b\/([a-zA-Z0-9-]+)/);
        if (match) {
            state.boardId = match[1];
            DOM.homeView.classList.remove('active');
            DOM.boardView.classList.add('active');
            initBoard();
        } else {
            DOM.homeView.classList.add('active');
            DOM.boardView.classList.remove('active');
        }
    }

    function initBoard() {
        console.log("Initializing board...");
        state.editorReadyPromise = initMonacoEditor();
        const storedUser = localStorage.getItem('huddleUser');
        if (storedUser) {
            state.user = JSON.parse(storedUser);
            DOM.modalOverlay.classList.add('hidden');
            connectWebSocket();
        } else {
            DOM.modalOverlay.classList.remove('hidden');
        }
    }

    // --- WebSocket ---
    function connectWebSocket() {
        if (state.ws) return;
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        state.ws = new WebSocket(`${protocol}//${window.location.host}/ws/${state.boardId}`);
        state.ws.onmessage = handleWebSocketMessage;
        state.ws.onopen = () => console.log("WebSocket Connected.");
        state.ws.onerror = (error) => console.error("WebSocket Error:", error);
    }
    
    function handleWebSocketMessage(event) {
        const msg = JSON.parse(event.data);
        state.programmaticChange = true;
        switch (msg.type) {
            case 'INITIAL_STATE': updateFullState(msg.payload, msg.clientId); break;
            case 'CODE_UPDATE': updateCode(msg.payload); break;
            case 'TASKS_UPDATE': updateTasks(msg.payload); break;
            case 'NOTES_UPDATE': updateNotes(msg.payload); break;
            case 'LINK_UPDATE': updateLink(msg.payload); break;
            case 'TEAM_UPDATE': updateTeam(msg.payload); break;
            case 'CHAT_MESSAGE': renderChatMessage(msg.payload); break;
        }
        setTimeout(() => state.programmaticChange = false, 50);
    }

    // --- State & UI Updates ---
    async function updateFullState(payload, clientId) {
        console.log("Received INITIAL_STATE. Awaiting editor readiness...");
        state.user.id = clientId;
        await state.editorReadyPromise;
        console.log("Editor ready. Proceeding with state updates.");
        
        updateCode(payload.contentCode);
        updateTasks(payload.contentTasks);
        updateNotes(payload.contentNotes);
        updateLink(payload.huddleLink);
        
        const currentTeam = payload.team || [];
        const userExists = currentTeam.some(member => member.id === state.user.id);
        
        if (!userExists) {
            const updatedTeam = [...currentTeam, state.user];
            sendMessage({ type: "TEAM_UPDATE", payload: updatedTeam });
            updateTeam(updatedTeam);
        } else {
            updateTeam(currentTeam);
        }
    }
    async function updateCode(payload) {
        await state.editorReadyPromise;
        if (state.editor.getValue() !== payload) state.editor.setValue(payload);
    }
    function updateTasks(payload) { state.tasks = payload || []; renderTasks(); }
    function updateNotes(payload) { if (document.activeElement !== DOM.notesEditor) { DOM.notesEditor.value = payload; } }
    function updateLink(payload) { if (document.activeElement !== DOM.huddleLinkInput) { DOM.huddleLinkInput.value = payload; } }
    function updateTeam(payload) { state.team = payload || []; renderTeam(); }

    function renderTasks() {
        DOM.taskBoard.innerHTML = '';
        state.tasks.forEach(task => {
            const assignee = state.team.find(m => m.name === task.assignee) || { name: 'Unassigned', color: '#ccc' };
            const taskEl = document.createElement('div');
            taskEl.className = `task-item ${task.completed ? 'completed' : ''}`;
            taskEl.dataset.id = task.id;
            taskEl.innerHTML = `<input type="checkbox" ${task.completed ? 'checked' : ''}><span>${escapeHTML(task.text)}</span><div class="assignee" style="background-color: ${assignee.color}" title="${assignee.name}">${escapeHTML(assignee.name.charAt(0).toUpperCase())}</div>`;
            DOM.taskBoard.appendChild(taskEl);
        });
        DOM.taskBoard.innerHTML += `<input type="text" class="task-input" placeholder="+ Add new task...">`;
        updateProgress();
    }
    function renderTeam() {
        DOM.teamList.innerHTML = '';
        state.team.forEach(member => {
            const isMe = member.id === state.user.id;
            const memberEl = document.createElement('li');
            memberEl.className = 'team-member';
            memberEl.innerHTML = `<div class="avatar" style="background-color: ${member.color}">${escapeHTML(member.name.charAt(0).toUpperCase())}</div><span>${escapeHTML(member.name)} ${isMe ? '(You)' : ''}</span>`;
            DOM.teamList.appendChild(memberEl);
        });
    }
    function renderChatMessage({ user, message }) {
        const msgEl = document.createElement('div');
        msgEl.innerHTML = `<strong style="color: ${user.color}">${escapeHTML(user.name)}:</strong> ${escapeHTML(message)}`;
        DOM.chatMessages.appendChild(msgEl);
        DOM.chatMessages.scrollTop = DOM.chatMessages.scrollHeight;
    }
    function updateProgress() {
        const completed = state.tasks.filter(t => t.completed).length;
        const total = state.tasks.length;
        const percent = total > 0 ? Math.round((completed / total) * 100) : 0;
        DOM.progressBar.style.width = `${percent}%`;
        DOM.progressText.textContent = `${percent}%`;
    }

    // --- Event Listeners ---
    function setupEventListeners() {
        DOM.createBoardBtn.onclick = () => { window.location.href = `/b/${Math.random().toString(36).substring(2, 12)}`; };
        DOM.navTabs.forEach(tab => tab.onclick = () => switchTab(tab.dataset.tab));
        DOM.userNameSetup.oninput = () => {
            const name = DOM.userNameSetup.value.trim() || "?";
            DOM.userAvatarSetup.style.backgroundColor = state.user.color;
            DOM.userAvatarSetup.textContent = name.charAt(0).toUpperCase();
        };
        DOM.joinHuddleBtn.onclick = handleJoinHuddle;
        DOM.editUserBtn.onclick = handleEditUser;
        DOM.huddleLinkInput.oninput = () => debounce('link', () => sendMessage({ type: "LINK_UPDATE", payload: DOM.huddleLinkInput.value }), 500);
        DOM.notesEditor.oninput = () => debounce('notes', () => sendMessage({ type: "NOTES_UPDATE", payload: DOM.notesEditor.value }), 500);
        DOM.taskBoard.addEventListener('click', handleTaskBoardClick);
        DOM.taskBoard.addEventListener('keydown', handleTaskInputKeydown);
        DOM.taskBoard.addEventListener('dblclick', handleTaskTextEdit);
        DOM.sendChatBtn.onclick = handleSendChat;
        DOM.chatInput.onkeydown = e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSendChat(); } };
        DOM.aiToolbarButtons.forEach(btn => btn.onclick = () => handleAIRequest(btn.dataset.aiAction));
        DOM.aiModalCloseBtn.onclick = () => DOM.modalOverlay.classList.add('hidden');
        DOM.joinVoiceBtn.onclick = handleVoiceButtonClick;
    }

    // --- Event Handlers ---
    function switchTab(tabId) {
        DOM.navTabs.forEach(t => t.classList.toggle('active', t.dataset.tab === tabId));
        DOM.mainPanels.forEach(p => p.classList.toggle('active', p.id === `${tabId}-panel`));
    }
    function handleJoinHuddle() {
        const name = DOM.userNameSetup.value.trim();
        if (!name) { alert("Please enter a name."); return; }
        state.user.name = name;
        localStorage.setItem('huddleUser', JSON.stringify(state.user));
        DOM.onboardingModal.classList.add('hidden');
        DOM.modalOverlay.classList.add('hidden');
        connectWebSocket();
    }
    function handleEditUser() {
        const newName = prompt("Enter your new name:", state.user.name);
        if (newName && newName.trim() !== "") {
            state.user.name = newName.trim();
            localStorage.setItem('huddleUser', JSON.stringify(state.user));
            const meInTeam = state.team.find(m => m.id === state.user.id);
            if(meInTeam) meInTeam.name = state.user.name;
            sendMessage({ type: "TEAM_UPDATE", payload: state.team });
        }
    }
    function handleTaskBoardClick(e) {
        const taskItem = e.target.closest('.task-item');
        if (!taskItem) return;
        const taskId = taskItem.dataset.id;
        const task = state.tasks.find(t => t.id === taskId);
        if (!task) return;
        let changed = false;
        if (e.target.type === 'checkbox') {
            task.completed = e.target.checked;
            changed = true;
        } else if (e.target.classList.contains('assignee')) {
            const teamMemberNames = state.team.map(m => m.name).join('\n');
            const newAssignee = prompt(`Assign task to a team member:\n(Enter a name from the list below)\n\n${teamMemberNames}`, task.assignee);
            if (newAssignee !== null) {
                const isValidMember = state.team.some(m => m.name.toLowerCase() === newAssignee.trim().toLowerCase());
                if (isValidMember || newAssignee.trim() === '') {
                    task.assignee = newAssignee.trim();
                    changed = true;
                } else {
                    alert(`'${newAssignee}' is not in the team. Please assign to an active member.`);
                }
            }
        }
        if (changed) { sendMessage({ type: "TASKS_UPDATE", payload: state.tasks }); }
    }
    function handleTaskTextEdit(e) {
        if (e.target.tagName !== 'SPAN') return;
        const taskItem = e.target.closest('.task-item');
        const taskId = taskItem.dataset.id;
        const task = state.tasks.find(t => t.id === taskId);
        if (task) {
            const newText = prompt("Edit task text:", task.text);
            if (newText !== null && newText.trim() !== '') {
                task.text = newText.trim();
                sendMessage({ type: "TASKS_UPDATE", payload: state.tasks });
            }
        }
    }
    function handleTaskInputKeydown(e) {
        if (e.target.classList.contains('task-input') && e.key === 'Enter') {
            const text = e.target.value.trim();
            if (text) {
                state.tasks.push({ id: `task-${Date.now()}`, text, completed: false, assignee: '' });
                sendMessage({ type: "TASKS_UPDATE", payload: state.tasks });
            }
        }
    }
    function handleSendChat() {
        const message = DOM.chatInput.value.trim();
        if (message) {
            const chatPayload = { user: state.user, message };
            sendMessage({ type: "CHAT_MESSAGE", payload: chatPayload });
            renderChatMessage(chatPayload);
            DOM.chatInput.value = '';
        }
    }
    async function handleAIRequest(action) {
        // The fix for 'marked is not defined' is to check if the library is loaded.
        if (typeof marked === 'undefined') {
            alert("Markdown library is still loading. Please try again in a moment.");
            return;
        }
        await state.editorReadyPromise;
        const code = state.editor.getValue();
        if (!code.trim()) { alert("There is no code to analyze."); return; }
        showModal(`AI Assistant: ${action.replace('_', ' ')}`, "Thinking...");
        try {
            const response = await fetch('/api/ai', {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ action, code, lang: DOM.languageSelect.value })
            });
            const data = await response.json();
            if (!response.ok) throw new Error(data.error);
            if (action === "refactor" || action === "add_comments") {
                state.editor.setValue(extractCodeFromMarkdown(data.result));
                DOM.modalOverlay.classList.add('hidden');
            } else {
                DOM.aiModalText.innerHTML = marked.parse(data.result);
            }
        } catch (error) { DOM.aiModalText.textContent = `Error: ${error.message}`; }
    }
    function handleVoiceButtonClick() {
        // The fix for the voice chat is to check if the Jitsi library is loaded.
        if (typeof JitsiMeetExternalAPI === 'undefined') {
            alert("Voice chat components are still loading. Please try again in a moment.");
            return;
        }
        if (state.jitsiApi) {
            state.jitsiApi.dispose();
            state.jitsiApi = null;
            DOM.voiceChatContainer.style.display = 'none';
            DOM.joinVoiceBtn.innerHTML = `<i class="bi bi-mic-fill"></i> Join Voice`;
        } else {
            const options = {
                roomName: `Huddle-Voice-${state.boardId}`,
                width: '100%',
                height: '100%',
                parentNode: DOM.voiceChatContainer,
                configOverwrite: { prejoinPageEnabled: false, startWithAudioMuted: false },
                interfaceConfigOverwrite: { TOOLBAR_BUTTONS: ['microphone', 'hangup'], DEFAULT_BACKGROUND: '#F3F4F6' }
            };
            state.jitsiApi = new JitsiMeetExternalAPI('8x8.vc', options);
            state.jitsiApi.executeCommand('displayName', state.user.name);
            DOM.voiceChatContainer.style.display = 'block';
            DOM.joinVoiceBtn.innerHTML = `<i class="bi bi-mic-mute-fill"></i> Leave Voice`;
        }
    }

    // --- Utilities ---
    function initMonacoEditor() {
        if (state.editorReadyPromise) return state.editorReadyPromise;
        state.editorReadyPromise = new Promise((resolve, reject) => {
            require.config({ paths: { 'vs': 'https://cdn.jsdelivr.net/npm/monaco-editor@0.44.0/min/vs' }});
            require(['vs/editor/editor.main'], () => {
                try {
                    state.editor = monaco.editor.create(DOM.editorContainer, {
                        theme: 'vs-dark', automaticLayout: true, fontSize: 14, wordWrap: 'on',
                        minimap: { enabled: false }, scrollBeyondLastLine: false,
                    });
                    state.editor.getModel().onDidChangeContent(() => {
                        if (state.programmaticChange) return;
                        debounce('code', () => sendMessage({ type: "CODE_UPDATE", payload: state.editor.getValue() }), 500);
                    });
                    DOM.languageSelect.onchange = () => monaco.editor.setModelLanguage(state.editor.getModel(), DOM.languageSelect.value);
                    console.log("Monaco Editor Initialized and Ready.");
                    resolve();
                } catch (error) { console.error("Failed to create Monaco editor:", error); reject(error); }
            }, (error) => { console.error("Failed to load Monaco editor scripts:", error); reject(error); });
        });
        return state.editorReadyPromise;
    }
    function showModal(title, text) {
        DOM.onboardingModal.classList.add('hidden');
        DOM.aiModal.classList.remove('hidden');
        DOM.aiModalTitle.textContent = title;
        DOM.aiModalText.innerHTML = text;
        DOM.modalOverlay.classList.remove('hidden');
    }
    function debounce(key, func, delay) { clearTimeout(state.debounceTimers[key]); state.debounceTimers[key] = setTimeout(func, delay); }
    function extractCodeFromMarkdown(markdown) {
        const match = markdown.match(/```(?:\w*\n)?([\s\S]*?)```/);
        // This is the fix for 'match.trim is not a function'
        return match && match[1] ? match[1].trim() : markdown;
    }
    function getRandomColor() { const c = ["#e53935","#d81b60","#8e24aa","#5e35b1","#3949ab","#1e88e5","#039be5","#00acc1","#00897b","#43a047","#f4511e"]; return c[Math.floor(Math.random()*c.length)]; }
    function escapeHTML(str) { const p = document.createElement('p'); p.appendChild(document.createTextNode(str)); return p.innerHTML; }
    function sendMessage(data) { if (state.ws && state.ws.readyState === WebSocket.OPEN) state.ws.send(JSON.stringify(data)); }

    // --- Start Application ---
    init();
});