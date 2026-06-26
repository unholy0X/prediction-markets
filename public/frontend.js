const { useState, useEffect } = React;
const {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} = Recharts;

// API Service
const API = {
  baseUrl: "http://localhost:3000/api",

  async request(endpoint, options = {}) {
    const token = localStorage.getItem("token");
    const headers = {
      "Content-Type": "application/json",
      ...(token && { Authorization: `Bearer ${token}` }),
      ...options.headers,
    };

    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      ...options,
      headers,
    });

    const data = await response.json();
    if (!response.ok) throw new Error(data.error || "Request failed");
    return data;
  },

  // Auth
  async register(email, password, name) {
    return this.request("/auth/register", {
      method: "POST",
      body: JSON.stringify({ email, password, name }),
    });
  },

  async login(email, password) {
    return this.request("/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    });
  },

  // Markets
  async getMarkets() {
    return this.request("/markets");
  },

  async getOrderBook(marketId) {
    return this.request(`/markets/${marketId}/orderbook`);
  },

  async getTrades(marketId) {
    return this.request(`/markets/${marketId}/trades`);
  },

  // Trading
  async placeOrder(order) {
    return this.request("/orders", {
      method: "POST",
      body: JSON.stringify(order),
    });
  },

  async getOrders() {
    return this.request("/orders");
  },

  async cancelOrder(orderId) {
    return this.request(`/orders/${orderId}`, {
      method: "DELETE",
    });
  },

  // Account
  async getBalance() {
    return this.request("/account/balance");
  },

  async getHistory() {
    return this.request("/account/history");
  },

  // Prediction Markets
  async getPredictions() {
    return this.request("/predictions");
  },

  async getPrediction(id) {
    return this.request(`/predictions/${id}`);
  },

  async getPredictionHistory(id) {
    return this.request(`/predictions/${id}/history`);
  },

  async quotePrediction(id, outcome, shares) {
    return this.request(`/predictions/${id}/quote`, {
      method: "POST",
      body: JSON.stringify({ outcome, shares }),
    });
  },

  async createPrediction(payload) {
    return this.request("/predictions", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },

  async tradePrediction(id, outcome, shares) {
    return this.request(`/predictions/${id}/trade`, {
      method: "POST",
      body: JSON.stringify({ outcome, shares }),
    });
  },

  async settlePrediction(id, winningOutcome) {
    return this.request(`/predictions/${id}/settle`, {
      method: "POST",
      body: JSON.stringify({ winningOutcome }),
    });
  },
};

// WebSocket Service
class WebSocketService {
  constructor() {
    this.ws = null;
    this.listeners = new Map();
  }

  connect() {
    this.ws = new WebSocket("ws://localhost:3000");

    this.ws.onmessage = (event) => {
      const data = JSON.parse(event.data);

      // Handle different message types
      if (data.type === "price_update") {
        // Update price in UI
        const listeners = this.listeners.get("price_update") || [];
        listeners.forEach((callback) => callback(data));
      } else if (data.type === "market_update") {
        const listeners = this.listeners.get("market_update") || [];
        listeners.forEach((callback) => callback(data.payload));
      } else {
        // Generic message handling
        const listeners = this.listeners.get(data.type) || [];
        listeners.forEach((callback) => callback(data));
      }
    };

    this.ws.onclose = () => {
      console.log("WebSocket disconnected");
      setTimeout(() => this.connect(), 5000);
    };
  }

  subscribe(marketId) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: "subscribe", marketId }));
    }
  }

  unsubscribe(marketId) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: "unsubscribe", marketId }));
    }
  }

  on(eventType, callback) {
    if (!this.listeners.has(eventType)) {
      this.listeners.set(eventType, []);
    }
    this.listeners.get(eventType).push(callback);
  }

  removeListener(eventType, callback) {
    const listeners = this.listeners.get(eventType) || [];
    const index = listeners.indexOf(callback);
    if (index !== -1) listeners.splice(index, 1);
  }
}

const wsService = new WebSocketService();

// Login Component
function Login({ onLogin }) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [isRegister, setIsRegister] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError("");

    try {
      if (isRegister) {
        const data = await API.register(email, password, name);
        localStorage.setItem("token", data.token);
        onLogin(data.user);
      } else {
        const data = await API.login(email, password);
        localStorage.setItem("token", data.token);
        onLogin(data.user);
      }
    } catch (err) {
      setError(err.message);
    }
  };

  return (
    <div className="login-container">
      <div className="login-box">
        <h2>{isRegister ? "Create Account" : "Welcome Back"}</h2>
        {error && <div className="error-message">{error}</div>}
        <form onSubmit={handleSubmit}>
          {isRegister && (
            <div className="form-group">
              <label>Name</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Your name"
              />
            </div>
          )}
          <div className="form-group">
            <label>Email</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="test@example.com"
              required
            />
          </div>
          <div className="form-group">
            <label>Password</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder={isRegister ? "Create password" : "Your password"}
              required
            />
          </div>
          <button type="submit" className="btn-primary">
            {isRegister ? "Register" : "Login"}
          </button>
        </form>
        <p className="toggle-auth">
          {isRegister ? "Already have an account?" : "Don't have an account?"}
          <button
            onClick={() => setIsRegister(!isRegister)}
            className="link-btn"
          >
            {isRegister ? "Login" : "Register"}
          </button>
        </p>
        <p className="test-credentials">Test: test@example.com / password123</p>
      </div>
    </div>
  );
}

// Market Card Component
function MarketCard({ market, onSelect }) {
  const changeColor = market.change24h >= 0 ? "positive" : "negative";

  return (
    <div className="market-card" onClick={() => onSelect(market)}>
      <div className="market-header">
        <h3>{market.name}</h3>
        {market.isPrediction && (
          <span className="prediction-badge">Prediction</span>
        )}
      </div>
      <div className="market-price">${market.currentPrice.toFixed(2)}</div>
      <div className={`market-change ${changeColor}`}>
        {market.change24h >= 0 ? "▲" : "▼"} {Math.abs(market.change24h)}%
      </div>
      <div className="market-volume">
        Vol: ${(market.volume24h / 1e6).toFixed(1)}M
      </div>
    </div>
  );
}

// Order Book Component
function OrderBook({ marketId }) {
  const [orderBook, setOrderBook] = useState({ bids: [], asks: [] });

  useEffect(() => {
    const fetchOrderBook = async () => {
      try {
        const data = await API.getOrderBook(marketId);
        setOrderBook(data);
      } catch (err) {
        console.error("Failed to fetch order book:", err);
      }
    };

    fetchOrderBook();
    const interval = setInterval(fetchOrderBook, 2000);

    return () => clearInterval(interval);
  }, [marketId]);

  return (
    <div className="order-book">
      <h3>Order Book</h3>
      <div className="order-book-header">
        <span>Price</span>
        <span>Size</span>
        <span>Total</span>
      </div>
      <div className="asks">
        {orderBook.asks
          .slice()
          .reverse()
          .map((ask, i) => (
            <div key={i} className="order-row ask">
              <span className="price">${ask.price.toFixed(2)}</span>
              <span>{ask.size.toFixed(4)}</span>
              <span>${(ask.price * ask.size).toFixed(2)}</span>
            </div>
          ))}
      </div>
      <div className="spread">
        Spread: $
        {(orderBook.asks[0]?.price - orderBook.bids[0]?.price).toFixed(2)}
      </div>
      <div className="bids">
        {orderBook.bids.map((bid, i) => (
          <div key={i} className="order-row bid">
            <span className="price">${bid.price.toFixed(2)}</span>
            <span>{bid.size.toFixed(4)}</span>
            <span>${(bid.price * bid.size).toFixed(2)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

// Trading Form Component
function TradingForm({ market, onOrderPlaced }) {
  const [side, setSide] = useState("buy");
  const [type, setType] = useState("limit");
  const [price, setPrice] = useState(market?.currentPrice || 0);
  const [quantity, setQuantity] = useState("");
  const [error, setError] = useState("");

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError("");

    try {
      await API.placeOrder({
        marketId: market.id,
        side,
        type,
        price: type === "limit" ? parseFloat(price) : undefined,
        quantity: parseFloat(quantity),
      });

      setQuantity("");
      onOrderPlaced();
    } catch (err) {
      setError(err.message);
    }
  };

  return (
    <div className="trading-form">
      <h3>Place Order</h3>
      <div className="order-type-selector">
        <button
          className={side === "buy" ? "active" : ""}
          onClick={() => setSide("buy")}
        >
          Buy
        </button>
        <button
          className={side === "sell" ? "active" : ""}
          onClick={() => setSide("sell")}
        >
          Sell
        </button>
      </div>
      <div className="order-type-selector">
        <button
          className={type === "limit" ? "active" : ""}
          onClick={() => setType("limit")}
        >
          Limit
        </button>
        <button
          className={type === "market" ? "active" : ""}
          onClick={() => setType("market")}
        >
          Market
        </button>
      </div>
      {error && <div className="error-message">{error}</div>}
      <form onSubmit={handleSubmit}>
        {type === "limit" && (
          <div className="form-group">
            <label>Price ($)</label>
            <input
              type="number"
              value={price}
              onChange={(e) => setPrice(e.target.value)}
              step="0.01"
              min="0"
              required
            />
          </div>
        )}
        <div className="form-group">
          <label>Quantity</label>
          <input
            type="number"
            value={quantity}
            onChange={(e) => setQuantity(e.target.value)}
            step="0.0001"
            min="0.0001"
            required
          />
        </div>
        <div className="form-group">
          <label>Total</label>
          <input
            type="text"
            value={`$${(parseFloat(price || 0) * parseFloat(quantity || 0)).toFixed(2)}`}
            disabled
          />
        </div>
        <button type="submit" className="btn-primary">
          Place {side === "buy" ? "Buy" : "Sell"} Order
        </button>
      </form>
    </div>
  );
}

// Price Chart Component
function PriceChart({ marketId }) {
  const [data, setData] = useState([]);

  useEffect(() => {
    // Generate sample price data
    const generateData = () => {
      const points = [];
      let price = 50000;
      for (let i = 30; i >= 0; i--) {
        price = price * (1 + (Math.random() - 0.5) * 0.02);
        points.push({
          time: i,
          price: price,
        });
      }
      setData(points);
    };

    generateData();
    const interval = setInterval(generateData, 10000);

    return () => clearInterval(interval);
  }, [marketId]);

  return (
    <div className="price-chart">
      <h3>Price Chart</h3>
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="time" />
          <YAxis domain={["auto", "auto"]} />
          <Tooltip />
          <Line type="monotone" dataKey="price" stroke="#8884d8" />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}

// ============================================================
// Prediction Markets
// ============================================================

// Client-side LMSR Yes-probability — mirrors the Go service so we can
// reconstruct a probability history by replaying past trades.
function lmsrYesPrice(qYes, qNo, b) {
  const m = Math.max(qYes, qNo);
  const eYes = Math.exp((qYes - m) / b);
  const eNo = Math.exp((qNo - m) / b);
  return eYes / (eYes + eNo);
}

// Horizontal Yes/No probability bar.
function ProbabilityMeter({ prices }) {
  const yes = (prices?.Yes ?? 0.5) * 100;
  const no = (prices?.No ?? 0.5) * 100;
  return (
    <div className="probability-meter">
      <div className="yes-probability" style={{ width: `${yes}%` }}>
        Yes {yes.toFixed(1)}%
      </div>
      <div className="no-probability" style={{ width: `${no}%` }}>
        No {no.toFixed(1)}%
      </div>
    </div>
  );
}

// Summary card for the markets grid.
function PredictionCard({ market, onSelect }) {
  return (
    <div className="prediction-card" onClick={() => onSelect(market)}>
      <div className="prediction-card-header">
        <h3>{market.question}</h3>
        <span className={`status-badge status-${market.status}`}>
          {market.status}
        </span>
      </div>
      <ProbabilityMeter prices={market.currentPrices} />
      <div className="prediction-meta">
        <span>Vol: ${market.volume24h.toFixed(2)}</span>
        <span>Liquidity: ${market.totalLiquidity.toFixed(2)}</span>
        <span>Expires: {new Date(market.expiryDate).toLocaleDateString()}</span>
      </div>
    </div>
  );
}

// Create-market form.
function CreateMarketForm({ onCreated }) {
  const [question, setQuestion] = useState("");
  const [description, setDescription] = useState("");
  const [expiryDate, setExpiryDate] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      await API.createPrediction({
        question,
        description,
        expiryDate: new Date(expiryDate).toISOString(),
      });
      setQuestion("");
      setDescription("");
      setExpiryDate("");
      onCreated();
    } catch (err) {
      setError(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="create-market-form">
      <h3>Create Prediction Market</h3>
      {error && <div className="error-message">{error}</div>}
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label>Question</label>
          <input
            type="text"
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
            placeholder="Will BTC reach $100k by Dec 31, 2026?"
            required
          />
        </div>
        <div className="form-group">
          <label>Description</label>
          <input
            type="text"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Optional context"
          />
        </div>
        <div className="form-group">
          <label>Expiry Date</label>
          <input
            type="datetime-local"
            value={expiryDate}
            onChange={(e) => setExpiryDate(e.target.value)}
            required
          />
        </div>
        <button type="submit" className="btn-primary" disabled={submitting}>
          {submitting ? "Creating..." : "Create Market"}
        </button>
      </form>
    </div>
  );
}

// Buy Yes/No shares, with a live cost quote.
function PredictionTradeForm({ market, onTraded }) {
  const [outcome, setOutcome] = useState("Yes");
  const [shares, setShares] = useState("");
  const [quote, setQuote] = useState(null);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  // Debounced live quote whenever outcome/shares change.
  useEffect(() => {
    const n = parseFloat(shares);
    if (!n || n <= 0) {
      setQuote(null);
      return;
    }
    let cancelled = false;
    const timer = setTimeout(() => {
      API.quotePrediction(market.id, outcome, n)
        .then((q) => !cancelled && setQuote(q))
        .catch(() => !cancelled && setQuote(null));
    }, 200);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [market.id, outcome, shares]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      await API.tradePrediction(market.id, outcome, parseFloat(shares));
      setShares("");
      setQuote(null);
      onTraded();
    } catch (err) {
      setError(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  if (market.status !== "active") {
    return (
      <div className="trading-form">
        <h3>Trading Closed</h3>
        <p>
          This market is <strong>{market.status}</strong>
          {market.winningOutcome ? ` — winner: ${market.winningOutcome}` : ""}.
        </p>
      </div>
    );
  }

  return (
    <div className="trading-form">
      <h3>Trade</h3>
      <div className="order-type-selector">
        <button
          type="button"
          className={outcome === "Yes" ? "active" : ""}
          onClick={() => setOutcome("Yes")}
        >
          Yes
        </button>
        <button
          type="button"
          className={outcome === "No" ? "active" : ""}
          onClick={() => setOutcome("No")}
        >
          No
        </button>
      </div>
      {error && <div className="error-message">{error}</div>}
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label>Shares</label>
          <input
            type="number"
            value={shares}
            onChange={(e) => setShares(e.target.value)}
            step="1"
            min="1"
            placeholder="100"
            required
          />
        </div>
        <div className="form-group">
          <label>Estimated Cost</label>
          <input
            type="text"
            value={
              quote
                ? `$${quote.totalCost.toFixed(2)} (avg $${quote.pricePerShare.toFixed(3)}/share)`
                : "—"
            }
            disabled
          />
        </div>
        <button
          type="submit"
          className="btn-primary"
          disabled={submitting || !quote}
        >
          {submitting ? "Trading..." : `Buy ${outcome}`}
        </button>
      </form>
    </div>
  );
}

// Reconstructs the Yes-probability trajectory by replaying trade history.
function ProbabilityChart({ market, history }) {
  const ordered = history.slice().reverse(); // history arrives newest-first
  let qYes = 0;
  let qNo = 0;
  const data = [{ t: "start", yes: 50 }];
  ordered.forEach((tr, i) => {
    if (tr.outcome === "Yes") qYes += tr.shares;
    else qNo += tr.shares;
    data.push({ t: `#${i + 1}`, yes: lmsrYesPrice(qYes, qNo, market.liquidityB) * 100 });
  });

  return (
    <div className="price-chart">
      <h3>Yes Probability History</h3>
      <ResponsiveContainer width="100%" height={260}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="t" />
          <YAxis domain={[0, 100]} tickFormatter={(v) => `${v}%`} />
          <Tooltip formatter={(v) => `${Number(v).toFixed(1)}%`} />
          <Line type="monotone" dataKey="yes" stroke="#16a34a" dot={false} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}

// Full market view: chart, position, trade form, admin settlement.
function PredictionDetail({ market, user, onClose, onChanged }) {
  const [history, setHistory] = useState([]);
  const [settleError, setSettleError] = useState("");

  useEffect(() => {
    let cancelled = false;
    API.getPredictionHistory(market.id)
      .then((h) => !cancelled && setHistory(h))
      .catch((err) => console.error("history load failed:", err));
    return () => {
      cancelled = true;
    };
    // Reload when volume changes (i.e. after a trade is recorded).
  }, [market.id, market.volume24h]);

  const myPosition = history
    .filter((t) => t.userId === user.id)
    .reduce((acc, t) => {
      acc[t.outcome] = (acc[t.outcome] || 0) + t.shares;
      return acc;
    }, {});

  const handleSettle = async (winningOutcome) => {
    setSettleError("");
    try {
      await API.settlePrediction(market.id, winningOutcome);
      onChanged();
    } catch (err) {
      setSettleError(err.message);
    }
  };

  return (
    <div className="market-detail">
      <div className="market-detail-header">
        <h2>{market.question}</h2>
        <button className="close-btn" onClick={onClose}>
          ×
        </button>
      </div>
      {market.description && (
        <p className="market-description">{market.description}</p>
      )}
      <ProbabilityMeter prices={market.currentPrices} />
      <div className="prediction-meta">
        <span>Status: {market.status}</span>
        <span>Volume: ${market.volume24h.toFixed(2)}</span>
        <span>Liquidity: ${market.totalLiquidity.toFixed(2)}</span>
        <span>Expires: {new Date(market.expiryDate).toLocaleString()}</span>
      </div>

      <div className="market-detail-content">
        <div className="left-panel">
          <ProbabilityChart market={market} history={history} />
          <div className="positions">
            <h3>My Position</h3>
            {Object.keys(myPosition).length === 0 ? (
              <p>No shares yet.</p>
            ) : (
              Object.entries(myPosition).map(([o, s]) => (
                <p key={o}>
                  <strong>{o}:</strong> {s} shares
                </p>
              ))
            )}
          </div>
        </div>
        <div className="right-panel">
          <PredictionTradeForm market={market} onTraded={onChanged} />
          {user.isAdmin && market.status !== "settled" && (
            <div className="settle-panel">
              <h3>Settle (Admin)</h3>
              {settleError && <div className="error-message">{settleError}</div>}
              <button className="btn-primary" onClick={() => handleSettle("Yes")}>
                Settle as Yes
              </button>
              <button
                className="btn-secondary"
                onClick={() => handleSettle("No")}
              >
                Settle as No
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// Predictions tab: create form + grid + detail.
function PredictionsView({ predictions, selected, user, onSelect, onClose, onChanged }) {
  return (
    <div className="predictions-view">
      <CreateMarketForm onCreated={onChanged} />
      <div className="predictions-grid">
        {predictions.length === 0 && <p>No prediction markets yet. Create one above.</p>}
        {predictions.map((market) => (
          <PredictionCard key={market.id} market={market} onSelect={onSelect} />
        ))}
      </div>
      {selected && (
        <PredictionDetail
          market={selected}
          user={user}
          onClose={onClose}
          onChanged={onChanged}
        />
      )}
    </div>
  );
}

// Main App Component
function App() {
  const [user, setUser] = useState(null);
  const [markets, setMarkets] = useState([]);
  const [selectedMarket, setSelectedMarket] = useState(null);
  const [balance, setBalance] = useState(0);
  const [orders, setOrders] = useState([]);
  const [predictions, setPredictions] = useState([]);
  const [selectedPredictionId, setSelectedPredictionId] = useState(null);
  const [activeTab, setActiveTab] = useState("markets");

  // Derive the selected prediction from the list so live WS updates flow into the detail view.
  const selectedPrediction =
    predictions.find((p) => p.id === selectedPredictionId) || null;

  useEffect(() => {
    wsService.connect();

    wsService.on("price_update", (data) => {
      setMarkets((prev) =>
        prev.map((m) =>
          m.id === data.marketId ? { ...m, currentPrice: data.price } : m,
        ),
      );
    });

    return () => {
      // Cleanup
    };
  }, []);

  useEffect(() => {
    if (user) {
      fetchMarkets();
      fetchBalance();
      fetchOrders();
      fetchPredictions();
    }
  }, [user]);

  // Real-time prediction updates straight from the Go prediction-service.
  useEffect(() => {
    if (!user) return;
    const ws = new WebSocket("ws://localhost:8082/ws");
    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data);
      if (
        ["price_update", "market_created", "market_settled"].includes(msg.type) &&
        msg.payload
      ) {
        setPredictions((prev) => {
          const exists = prev.some((p) => p.id === msg.payload.id);
          return exists
            ? prev.map((p) => (p.id === msg.payload.id ? msg.payload : p))
            : [msg.payload, ...prev];
        });
      }
    };
    ws.onerror = () => {}; // service may be down; list still works via REST
    return () => ws.close();
  }, [user]);

  const fetchMarkets = async () => {
    try {
      const data = await API.getMarkets();
      setMarkets(data);
    } catch (err) {
      console.error("Failed to fetch markets:", err);
    }
  };

  const fetchBalance = async () => {
    try {
      const data = await API.getBalance();
      setBalance(data.balance);
    } catch (err) {
      console.error("Failed to fetch balance:", err);
    }
  };

  const fetchOrders = async () => {
    try {
      const data = await API.getOrders();
      setOrders(data);
    } catch (err) {
      console.error("Failed to fetch orders:", err);
    }
  };

  const fetchPredictions = async () => {
    try {
      const data = await API.getPredictions();
      setPredictions(data);
    } catch (err) {
      console.error("Failed to fetch predictions:", err);
    }
  };

  // Refresh after any market-changing action (create/trade/settle).
  const refreshPredictions = async () => {
    await fetchPredictions();
    await fetchBalance();
  };

  const handleLogout = () => {
    localStorage.removeItem("token");
    setUser(null);
  };

  if (!user) {
    return <Login onLogin={setUser} />;
  }

  return (
    <div className="app">
      <header className="header">
        <h1>Trading Prediction Platform</h1>
        <div className="user-info">
          <span>Welcome, {user.name}</span>
          <span className="balance">Balance: ${balance.toFixed(2)}</span>
          <button onClick={handleLogout} className="logout-btn">
            Logout
          </button>
        </div>
      </header>

      <nav className="nav-tabs">
        <button
          className={activeTab === "markets" ? "active" : ""}
          onClick={() => setActiveTab("markets")}
        >
          Markets
        </button>
        <button
          className={activeTab === "predictions" ? "active" : ""}
          onClick={() => setActiveTab("predictions")}
        >
          Predictions
        </button>
        <button
          className={activeTab === "orders" ? "active" : ""}
          onClick={() => setActiveTab("orders")}
        >
          My Orders
        </button>
        <button
          className={activeTab === "account" ? "active" : ""}
          onClick={() => setActiveTab("account")}
        >
          Account
        </button>
      </nav>

      <main className="main-content">
        {activeTab === "markets" && (
          <div className="markets-view">
            <div className="markets-grid">
              {markets.map((market) => (
                <MarketCard
                  key={market.id}
                  market={market}
                  onSelect={() => {
                    setSelectedMarket(market);
                    wsService.subscribe(market.id);
                  }}
                />
              ))}
            </div>

            {selectedMarket && (
              <div className="market-detail">
                <div className="market-detail-header">
                  <h2>{selectedMarket.name}</h2>
                  <button
                    className="close-btn"
                    onClick={() => {
                      wsService.unsubscribe(selectedMarket.id);
                      setSelectedMarket(null);
                    }}
                  >
                    ×
                  </button>
                </div>

                <div className="market-detail-content">
                  <div className="left-panel">
                    <PriceChart marketId={selectedMarket.id} />
                    <OrderBook marketId={selectedMarket.id} />
                  </div>
                  <div className="right-panel">
                    <TradingForm
                      market={selectedMarket}
                      onOrderPlaced={() => {
                        fetchOrders();
                        fetchBalance();
                      }}
                    />
                  </div>
                </div>
              </div>
            )}
          </div>
        )}

        {activeTab === "predictions" && (
          <PredictionsView
            predictions={predictions}
            selected={selectedPrediction}
            user={user}
            onSelect={(market) => setSelectedPredictionId(market.id)}
            onClose={() => setSelectedPredictionId(null)}
            onChanged={refreshPredictions}
          />
        )}

        {activeTab === "orders" && (
          <div className="orders-view">
            <h2>My Orders</h2>
            <table className="orders-table">
              <thead>
                <tr>
                  <th>Market</th>
                  <th>Side</th>
                  <th>Type</th>
                  <th>Price</th>
                  <th>Quantity</th>
                  <th>Filled</th>
                  <th>Status</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {orders.map((order) => {
                  const market = markets.find((m) => m.id === order.marketId);
                  return (
                    <tr key={order.id}>
                      <td>{market?.symbol || order.marketId}</td>
                      <td className={order.side}>{order.side}</td>
                      <td>{order.type}</td>
                      <td>${order.price?.toFixed(2)}</td>
                      <td>{order.quantity}</td>
                      <td>{order.filledQuantity}</td>
                      <td>
                        <span className={`status-${order.status}`}>
                          {order.status}
                        </span>
                      </td>
                      <td>
                        {order.status === "open" && (
                          <button
                            onClick={async () => {
                              await API.cancelOrder(order.id);
                              fetchOrders();
                            }}
                            className="cancel-btn"
                          >
                            Cancel
                          </button>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}

        {activeTab === "account" && (
          <div className="account-view">
            <h2>Account Overview</h2>
            <div className="account-info">
              <div className="info-card">
                <h3>Profile</h3>
                <p>
                  <strong>Name:</strong> {user.name}
                </p>
                <p>
                  <strong>Email:</strong> {user.email}
                </p>
                <p>
                  <strong>Member since:</strong>{" "}
                  {new Date(user.createdAt).toLocaleDateString()}
                </p>
              </div>

              <div className="info-card">
                <h3>Balance</h3>
                <p className="large-balance">${balance.toFixed(2)}</p>
                <p>
                  <strong>Available:</strong> ${balance.toFixed(2)}
                </p>
              </div>

              <div className="info-card">
                <h3>Statistics</h3>
                <p>
                  <strong>Total Orders:</strong> {orders.length}
                </p>
                <p>
                  <strong>Open Orders:</strong>{" "}
                  {orders.filter((o) => o.status === "open").length}
                </p>
                <p>
                  <strong>Filled Orders:</strong>{" "}
                  {orders.filter((o) => o.status === "filled").length}
                </p>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}

// Render the app
const root = ReactDOM.createRoot(document.getElementById("root"));
root.render(<App />);
