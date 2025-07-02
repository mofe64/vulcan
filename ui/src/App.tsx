import "./App.css";

function App() {
  return (
    <div
      style={{
        minHeight: "100vh",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        background: "linear-gradient(135deg, #0f2027, #2c5364)",
        color: "#fff",
        fontFamily: "Inter, sans-serif",
      }}
    >
      <h1
        style={{
          fontSize: "3rem",
          fontWeight: 700,
          marginBottom: "1rem",
          letterSpacing: "2px",
          textShadow: "0 2px 16px rgba(0,0,0,0.2)",
        }}
      >
        Hello World from Vulkan
      </h1>
      <p
        style={{
          fontSize: "1.25rem",
          opacity: 0.8,
        }}
      >
        Welcome to your new React + TypeScript Vite app!
      </p>
    </div>
  );
}

export default App;
