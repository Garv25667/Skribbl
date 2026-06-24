// ============================================================
// CONFIG
// ============================================================
const API_URL = 'http://localhost:3223';
const WS_URL = 'ws://localhost:3223';

// ============================================================
// STATE
// ============================================================
let token = null;
let username = '';
let ws = null;
let roomId = '';
let isDrawer = false;
let isHost = false;
let currentWord = '';
let players = [];
let timerInterval = null;
let timeLeft = 80;

const drawState = {
    drawing: false,
    color: '#000000',
    lineWidth: 3,
    eraser: false,
    prevX: 0,
    prevY: 0
};

const COLORS = [
    '#000000', '#ffffff', '#808080', '#c0392b', '#e74c3c', '#e67e22',
    '#f1c40f', '#2ecc71', '#27ae60', '#3498db', '#2980b9', '#8e44ad',
    '#9b59b6', '#1abc9c', '#e91e63', '#795548'
];

// ============================================================
// DOM REFS
// ============================================================
const $ = (id) => document.getElementById(id);

const screens = {
    login: $('screen-login'),
    lobby: $('screen-lobby'),
    game: $('screen-game')
};

// ============================================================
// SCREEN MANAGEMENT
// ============================================================
function showScreen(name) {
    Object.values(screens).forEach(s => s.classList.remove('active'));
    screens[name].classList.add('active');
}

// ============================================================
// LOGIN
// ============================================================
$('btn-login').addEventListener('click', doLogin);
$('input-username').addEventListener('keydown', (e) => {
    if (e.key === 'Enter') doLogin();
});

async function doLogin() {
    username = $('input-username').value.trim();
    if (!username) {
        $('login-error').textContent = 'Enter a name!';
        return;
    }
    $('login-error').textContent = '';

    try {
        const res = await fetch(`${API_URL}/register?username=${encodeURIComponent(username)}`);
        if (!res.ok) {
            $('login-error').textContent = await res.text();
            return;
        }
        token = await res.text();
        $('display-username').textContent = username;
        showScreen('lobby');
        $('input-room').focus();
    } catch (e) {
        $('login-error').textContent = 'Cannot connect to server';
    }
}

// ============================================================
// LOBBY
// ============================================================
$('btn-join').addEventListener('click', doJoin);
$('input-room').addEventListener('keydown', (e) => {
    if (e.key === 'Enter') doJoin();
});

$('btn-create').addEventListener('click', () => {
    const code = Math.random().toString(36).substring(2, 8).toUpperCase();
    $('input-room').value = code;
    doJoin();
});

function doJoin() {
    roomId = $('input-room').value.trim();
    if (!roomId) {
        $('lobby-error').textContent = 'Enter a room code';
        return;
    }
    $('lobby-error').textContent = '';
    connectToRoom(roomId);
}

// ============================================================
// WEBSOCKET CONNECTION
// ============================================================
function connectToRoom(room) {
    ws = new WebSocket(`${WS_URL}/?room=${encodeURIComponent(room)}&token=${encodeURIComponent(token)}`);

    ws.onopen = () => {
        showScreen('game');
        $('game-word').textContent = `Room: ${room}`;
        $('game-round').textContent = 'Waiting...';
        $('game-timer').textContent = '⏱ --';
        clearCanvas();
    };

    ws.onclose = () => {
        addSystemMsg('Disconnected from server.');
    };

    ws.onerror = () => {
        $('lobby-error').textContent = 'WebSocket connection failed';
    };

    ws.onmessage = (event) => {
        const msg = JSON.parse(event.data);
        handleMessage(msg);
    };
}

// ============================================================
// MESSAGE ROUTER
// ============================================================
function handleMessage(msg) {
    const handler = {
        'room_info':      () => onRoomInfo(msg.Data),
        'player_joined':  () => onPlayerJoined(msg.Data),
        'player_left':    () => onPlayerLeft(msg.Data),
        'game_started':   () => onGameStarted(msg.Data),
        'your_turn':      () => onYourTurn(msg.Data),
        'word_hint':      () => onWordHint(msg.Data),
        'correct_guess':  () => onCorrectGuess(msg.Data),
        'turn_ended':     () => onTurnEnded(msg.Data),
        'game_ended':     () => onGameEnded(msg.Data),
        'broadcast':      () => onBroadcast(msg.Data),
    }[msg.MsgType];

    if (handler) handler();
}

// ============================================================
// EVENT HANDLERS
// ============================================================
function onRoomInfo(data) {
    players = (data.CurrentPlayers || []).map(p => ({
        ...p,
        isDrawing: false,
        guessedCorrectly: false
    }));

    // Identify self and host status
    const me = players.find(p => p.Username === username);
    if (me) isHost = me.IsHost;

    renderPlayers();
    $('btn-start').style.display = isHost ? 'block' : 'none';
}

function onPlayerJoined(data) {
    // Avoid duplicates
    if (players.find(p => p.PlayerID === data.PlayerID)) return;

    players.push({
        PlayerID: data.PlayerID,
        Username: data.Username,
        IsHost: false,
        isDrawing: false,
        guessedCorrectly: false
    });
    renderPlayers();
    addSystemMsg(`${data.Username} joined`);
}

function onPlayerLeft(data) {
    const p = players.find(p => p.PlayerID === data.PlayerID);
    const name = p ? p.Username : '???';
    players = players.filter(p => p.PlayerID !== data.PlayerID);
    renderPlayers();
    addSystemMsg(`${name} left`);
}

function onGameStarted(data) {
    $('game-round').textContent = `Round ${data.Round} / ${data.MaxRounds}`;
    $('btn-start').style.display = 'none';

    // Mark drawer
    players.forEach(p => {
        p.isDrawing = (p.PlayerID === data.DrawerID);
        p.guessedCorrectly = false;
    });
    renderPlayers();
    startTimer(80);
}

function onYourTurn(data) {
    isDrawer = true;
    currentWord = data.Word;
    $('game-word').textContent = data.Word;
    $('game-toolbar').style.display = 'flex';
    $('input-guess').disabled = true;
    $('input-guess').placeholder = "You're drawing!";
    $('btn-guess').disabled = true;
    clearCanvas();
    addSystemMsg(`Your turn! Draw: ${data.Word}`);
}

function onWordHint(data) {
    isDrawer = false;
    currentWord = '';
    $('game-word').textContent = data.Hint;
    $('game-toolbar').style.display = 'none';
    $('input-guess').disabled = false;
    $('input-guess').placeholder = 'Type your guess...';
    $('btn-guess').disabled = false;
    $('input-guess').focus();
    clearCanvas();
}

function onCorrectGuess(data) {
    const p = players.find(p => p.PlayerID === data.PlayerID);
    if (p) {
        p.guessedCorrectly = true;
        addSystemMsg(`${p.Username} guessed correctly! 🎉`);
    }
    renderPlayers();
}

function onTurnEnded(data) {
    stopTimer();
    $('game-word').textContent = `Word was: ${data.Word}`;
    addSystemMsg(`Turn ended — the word was "${data.Word}"`);

    // Brief pause before next turn resets everything
    $('game-toolbar').style.display = 'none';
    isDrawer = false;
}

function onGameEnded(data) {
    stopTimer();
    isDrawer = false;
    $('game-toolbar').style.display = 'none';
    $('game-word').textContent = '🏆 Game Over!';
    $('game-round').textContent = 'Finished';
    $('game-timer').textContent = '⏱ --';
    $('input-guess').disabled = true;

    addEndMsg('━━━ Game Over ━━━');

    if (data.Scores) {
        const sorted = Object.entries(data.Scores)
            .map(([id, score]) => {
                const p = players.find(p => p.PlayerID === id);
                return { name: p ? p.Username : id, score };
            })
            .sort((a, b) => b.score - a.score);

        const medals = ['🥇', '🥈', '🥉'];
        sorted.forEach((entry, i) => {
            addEndMsg(`${medals[i] || '  '} ${entry.name}: ${entry.score} pts`);
        });
    }

    // Let host restart
    if (isHost) {
        $('btn-start').style.display = 'block';
    }
}

function onBroadcast(data) {
    // broadcast Data is a JSON string (double-serialized by server)
    const stroke = typeof data === 'string' ? JSON.parse(data) : data;

    if (stroke.t === 'c') {
        clearCanvas();
        return;
    }

    if (stroke.t === 'd') {
        const canvas = $('game-canvas');
        const ctx = canvas.getContext('2d');
        ctx.strokeStyle = stroke.c;
        ctx.lineWidth = stroke.w;
        ctx.lineCap = 'round';
        ctx.lineJoin = 'round';
        ctx.beginPath();
        ctx.moveTo(stroke.px, stroke.py);
        ctx.lineTo(stroke.x, stroke.y);
        ctx.stroke();
    }
}

// ============================================================
// TIMER
// ============================================================
function startTimer(seconds) {
    stopTimer();
    timeLeft = seconds;
    updateTimerDisplay();
    timerInterval = setInterval(() => {
        timeLeft--;
        updateTimerDisplay();
        if (timeLeft <= 0) stopTimer();
    }, 1000);
}

function stopTimer() {
    if (timerInterval) {
        clearInterval(timerInterval);
        timerInterval = null;
    }
}

function updateTimerDisplay() {
    const el = $('game-timer');
    el.textContent = `⏱ ${timeLeft}`;
    el.classList.toggle('urgent', timeLeft <= 10);
}

// ============================================================
// PLAYER LIST
// ============================================================
function renderPlayers() {
    const list = $('player-list');
    list.innerHTML = '';

    players.forEach(p => {
        const li = document.createElement('li');
        if (p.isDrawing) li.classList.add('drawing');

        const nameSpan = document.createElement('span');
        nameSpan.classList.add('player-name');
        let label = p.Username;
        if (p.IsHost) label = '👑 ' + label;
        if (p.isDrawing) label += ' 🎨';
        nameSpan.textContent = label;
        li.appendChild(nameSpan);

        if (p.guessedCorrectly) {
            const badge = document.createElement('span');
            badge.classList.add('badge');
            badge.textContent = '✅';
            li.appendChild(badge);
        }

        list.appendChild(li);
    });
}

// ============================================================
// CHAT
// ============================================================
function addSystemMsg(text) {
    const div = document.createElement('div');
    div.classList.add('msg', 'system');
    div.textContent = text;
    appendChat(div);
}

function addEndMsg(text) {
    const div = document.createElement('div');
    div.classList.add('msg', 'system', 'end');
    div.textContent = text;
    appendChat(div);
}

function addPlayerMsg(author, text) {
    const div = document.createElement('div');
    div.classList.add('msg');
    const authorSpan = document.createElement('span');
    authorSpan.classList.add('author');
    authorSpan.textContent = author + ': ';
    div.appendChild(authorSpan);
    div.appendChild(document.createTextNode(text));
    appendChat(div);
}

function appendChat(el) {
    const messages = $('chat-messages');
    messages.appendChild(el);
    messages.scrollTop = messages.scrollHeight;
}

// ============================================================
// GUESS INPUT
// ============================================================
$('btn-guess').addEventListener('click', sendGuess);
$('input-guess').addEventListener('keydown', (e) => {
    if (e.key === 'Enter') sendGuess();
});

function sendGuess() {
    const input = $('input-guess');
    const guess = input.value.trim();
    if (!guess || !ws || isDrawer) return;

    ws.send(JSON.stringify({ MsgType: 'Guess', Data: guess }));
    addPlayerMsg(username, guess);
    input.value = '';
    input.focus();
}

// ============================================================
// START GAME
// ============================================================
$('btn-start').addEventListener('click', () => {
    if (!ws) return;
    ws.send(JSON.stringify({ MsgType: 'StartGame', Data: '' }));
    $('btn-start').style.display = 'none';
});

// ============================================================
// CANVAS — DRAWING
// ============================================================
const canvas = $('game-canvas');
const ctx = canvas.getContext('2d');

// Initialize color picker
const colorPicker = $('color-picker');
COLORS.forEach((color, i) => {
    const btn = document.createElement('button');
    btn.classList.add('color-btn');
    if (i === 0) btn.classList.add('active');
    btn.style.backgroundColor = color;
    if (color === '#ffffff') {
        btn.style.border = '2px solid #555';
    }
    btn.addEventListener('click', () => {
        document.querySelectorAll('.color-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        drawState.color = color;
        drawState.eraser = false;
        $('btn-eraser').classList.remove('active');
    });
    colorPicker.appendChild(btn);
});

// Brush sizes
document.querySelectorAll('.brush-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        document.querySelectorAll('.brush-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        drawState.lineWidth = parseInt(btn.dataset.size);
    });
});

// Eraser toggle
$('btn-eraser').addEventListener('click', () => {
    drawState.eraser = !drawState.eraser;
    $('btn-eraser').classList.toggle('active', drawState.eraser);
    if (drawState.eraser) {
        document.querySelectorAll('.color-btn').forEach(b => b.classList.remove('active'));
    }
});

// Clear canvas
$('btn-clear').addEventListener('click', () => {
    clearCanvas();
    if (ws) {
        ws.send(JSON.stringify({ MsgType: 'broadcast', Data: JSON.stringify({ t: 'c' }) }));
    }
});

function clearCanvas() {
    ctx.fillStyle = '#ffffff';
    ctx.fillRect(0, 0, canvas.width, canvas.height);
}
clearCanvas();

// ============================================================
// CANVAS — MOUSE / TOUCH EVENTS
// ============================================================
function getCanvasPos(e) {
    const rect = canvas.getBoundingClientRect();
    const scaleX = canvas.width / rect.width;
    const scaleY = canvas.height / rect.height;

    if (e.touches && e.touches.length > 0) {
        return {
            x: (e.touches[0].clientX - rect.left) * scaleX,
            y: (e.touches[0].clientY - rect.top) * scaleY
        };
    }
    return {
        x: (e.clientX - rect.left) * scaleX,
        y: (e.clientY - rect.top) * scaleY
    };
}

function onDrawStart(e) {
    if (!isDrawer) return;
    e.preventDefault();
    drawState.drawing = true;
    const pos = getCanvasPos(e);
    drawState.prevX = pos.x;
    drawState.prevY = pos.y;
}

function onDrawMove(e) {
    if (!isDrawer || !drawState.drawing) return;
    e.preventDefault();
    const pos = getCanvasPos(e);
    drawLine(drawState.prevX, drawState.prevY, pos.x, pos.y);
    drawState.prevX = pos.x;
    drawState.prevY = pos.y;
}

function onDrawEnd() {
    drawState.drawing = false;
}

canvas.addEventListener('mousedown', onDrawStart);
canvas.addEventListener('mousemove', onDrawMove);
canvas.addEventListener('mouseup', onDrawEnd);
canvas.addEventListener('mouseleave', onDrawEnd);

canvas.addEventListener('touchstart', onDrawStart, { passive: false });
canvas.addEventListener('touchmove', onDrawMove, { passive: false });
canvas.addEventListener('touchend', onDrawEnd);

function drawLine(px, py, x, y) {
    const color = drawState.eraser ? '#ffffff' : drawState.color;
    const width = drawState.eraser ? 24 : drawState.lineWidth;

    // Draw locally
    ctx.strokeStyle = color;
    ctx.lineWidth = width;
    ctx.lineCap = 'round';
    ctx.lineJoin = 'round';
    ctx.beginPath();
    ctx.moveTo(px, py);
    ctx.lineTo(x, y);
    ctx.stroke();

    // Send to others
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({
            MsgType: 'broadcast',
            Data: JSON.stringify({
                t: 'd',
                px: Math.round(px),
                py: Math.round(py),
                x: Math.round(x),
                y: Math.round(y),
                c: color,
                w: width
            })
        }));
    }
}
