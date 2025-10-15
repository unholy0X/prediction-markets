const express = require('express');
const cors = require('cors');
const http = require('http');
const WebSocket = require('ws');
const bcrypt = require('bcryptjs');
const jwt = require('jsonwebtoken');
const { v4: uuidv4 } = require('uuid');
const axios = require('axios');
const cookieParser = require('cookie-parser');


const app = express();
const server = http.createServer(app);
const wss = new WebSocket.Server({ server });

// Middleware
app.use(cors({
  origin: ['http://localhost:3000', 'http://localhost:3001', 'http://127.0.0.1:3000', 'http://127.0.0.1:3001'],
  credentials: true
}));
app.use(express.json());
app.use(express.static(__dirname));
app.use(cookieParser());

const user = require('./routes/userRoute');
const product = require('./routes/productRoute');
const order = require('./routes/orderRoute');
const payment = require('./routes/paymentRoute');

app.use('/api/user', user);
app.use('/api/product', product);
app.use('/api/order', order);
app.use('/api/payment', payment);

// Constants
const JWT_SECRET = 'your-super-secret-jwt-key-for-testing-only';
const PORT = 3000;

// Go service URLs
const GO_ENGINE_URL = 'http://localhost:8080';
const GO_MARKET_URL = 'http://localhost:8081';

// In-memory database (for testing)
const users = [];

// Create test user
const testUser = {
  id: 'test-user-1',
  email: 'test@example.com',
  name: 'Test User',
  password: bcrypt.hashSync('password123', 10),
  balance: 10000,
  createdAt: new Date().toISOString()
};
users.push(testUser);

// Authentication middleware
function authenticateToken(req, res, next) {
  const authHeader = req.headers['authorization'];
  const token = authHeader && authHeader.split(' ')[1];
  
  if (!token) {
    return res.status(401).json({ error: 'Authentication required' });
  }
  
  jwt.verify(token, JWT_SECRET, (err, user) => {
    if (err) {
      return res.status(403).json({ error: 'Invalid or expired token' });
    }
    req.user = user;
    next();
  });
}

// Proxy to Go services
app.get('/api/markets', async (req, res) => {
  try {
    const response = await axios.get(`${GO_MARKET_URL}/api/markets`);
    res.json(response.data);
  } catch (error) {
    console.error('Error fetching markets:', error.message);
    // Fallback to mock data if Go service is down
    res.json([
      {
        id: '1',
        name: 'BTC/USD',
        symbol: 'BTC-USD',
        currentPrice: 52345.67,
        change24h: 2.34,
        volume24h: 1250000000,
        high24h: 53456.78,
        low24h: 51234.56
      },
      {
        id: '2',
        name: 'ETH/USD',
        symbol: 'ETH-USD',
        currentPrice: 3123.45,
        change24h: -1.23,
        volume24h: 850000000,
        high24h: 3245.67,
        low24h: 3098.76
      }
    ]);
  }
});

app.get('/api/markets/:id', async (req, res) => {
  try {
    const response = await axios.get(`${GO_MARKET_URL}/api/markets/${req.params.id}`);
    res.json(response.data);
  } catch (error) {
    res.status(404).json({ error: 'Market not found' });
  }
});

app.get('/api/markets/:id/orderbook', async (req, res) => {
  try {
    const response = await axios.get(`${GO_ENGINE_URL}/api/orderbook/${req.params.id}`);
    res.json(response.data);
  } catch (error) {
    // Fallback mock data
    res.json({
      bids: Array(10).fill().map((_, i) => ({
        price: 50000 - i * 10,
        size: Math.random() * 5
      })),
      asks: Array(10).fill().map((_, i) => ({
        price: 50000 + i * 10,
        size: Math.random() * 5
      }))
    });
  }
});

app.get('/api/markets/:id/candles', async (req, res) => {
  try {
    const response = await axios.get(`${GO_MARKET_URL}/api/markets/${req.params.id}/candles`);
    res.json(response.data);
  } catch (error) {
    // Fallback mock data
    const candles = [];
    const now = Date.now();
    for (let i = 0; i < 30; i++) {
      candles.push({
        time: now - i * 3600000,
        open: 50000 + Math.random() * 1000,
        high: 51000 + Math.random() * 1000,
        low: 49000 + Math.random() * 1000,
        close: 50500 + Math.random() * 1000,
        volume: 1000000 + Math.random() * 500000
      });
    }
    res.json(candles);
  }
});

app.post('/api/orders', authenticateToken, async (req, res) => {
  try {
    const order = {
      ...req.body,
      user_id: req.user.id,
      id: uuidv4(),
      created_at: new Date().toISOString()
    };
    
    const response = await axios.post(`${GO_ENGINE_URL}/api/orders`, order);
    res.json(response.data);
  } catch (error) {
    console.error('Error placing order:', error.message);
    res.status(500).json({ error: 'Failed to place order' });
  }
});

app.get('/api/trades', async (req, res) => {
  try {
    const response = await axios.get(`${GO_ENGINE_URL}/api/trades`);
    res.json(response.data);
  } catch (error) {
    res.json([]);
  }
});

// Auth routes
app.post('/api/auth/register', async (req, res) => {
  const { email, password, name } = req.body;
  
  if (!email || !password) {
    return res.status(400).json({ error: 'Email and password required' });
  }
  
  if (users.find(u => u.email === email)) {
    return res.status(400).json({ error: 'User already exists' });
  }
  
  const hashedPassword = await bcrypt.hash(password, 10);
  const user = {
    id: uuidv4(),
    email,
    name: name || email.split('@')[0],
    password: hashedPassword,
    balance: 10000,
    createdAt: new Date().toISOString()
  };
  
  users.push(user);
  
  const token = jwt.sign({ id: user.id, email: user.email }, JWT_SECRET);
  
  res.json({
    token,
    user: {
      id: user.id,
      email: user.email,
      name: user.name,
      balance: user.balance
    }
  });
});

app.post('/api/auth/login', async (req, res) => {
  const { email, password } = req.body;
  
  const user = users.find(u => u.email === email);
  if (!user) {
    return res.status(401).json({ error: 'Invalid credentials' });
  }
  
  const validPassword = await bcrypt.compare(password, user.password);
  if (!validPassword) {
    return res.status(401).json({ error: 'Invalid credentials' });
  }
  
  const token = jwt.sign({ id: user.id, email: user.email }, JWT_SECRET);
  
  res.json({
    token,
    user: {
      id: user.id,
      email: user.email,
      name: user.name,
      balance: user.balance
    }
  });
});

app.get('/api/account/balance', authenticateToken, (req, res) => {
  const user = users.find(u => u.id === req.user.id);
  res.json({ balance: user.balance });
});

// Health check
app.get('/api/health', (req, res) => {
  res.json({ 
    status: 'ok', 
    timestamp: Date.now(),
    services: {
      backend: 'running',
      goEngine: 'checking...',
      goMarket: 'checking...'
    }
  });
});

// WebSocket connection for real-time updates
wss.on('connection', (ws) => {
  console.log('WebSocket client connected');
  
  ws.send(JSON.stringify({
    type: 'connected',
    message: 'Connected to WebSocket server'
  }));
  
  // Proxy WebSocket messages to Go services
  ws.on('message', (message) => {
    try {
      const data = JSON.parse(message);
      // Handle subscription logic here
      console.log('Received:', data);
    } catch (error) {
      console.error('WebSocket message error:', error);
    }
  });
  
  ws.on('close', () => {
    console.log('WebSocket client disconnected');
  });
});

// Start server
server.listen(PORT, () => {
  console.log(`✅ Backend server running on http://localhost:${PORT}`);
  console.log(`✅ WebSocket server on ws://localhost:${PORT}`);
  console.log(`✅ Test user: test@example.com / password123`);
  console.log(`\n📡 Go Services (if running):`);
  console.log(`   Order Matching Engine: http://localhost:8080`);
  console.log(`   Market Data Service: http://localhost:8081`);
  console.log(`\n🚀 Frontend: http://localhost:3001`);
});