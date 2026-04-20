import { Component, StrictMode, type ErrorInfo, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import "./theme.css";

// ErrorBoundary turns render-time exceptions into a visible panel
// instead of a blank tree. Plain styling, for wireframe-era clarity.
class ErrorBoundary extends Component<{ children: ReactNode }, { err: Error | null }> {
  state = { err: null as Error | null };
  static getDerivedStateFromError(err: Error) {
    return { err };
  }
  componentDidCatch(err: Error, info: ErrorInfo) {
    // eslint-disable-next-line no-console
    console.error("react crash", err, info.componentStack);
  }
  render() {
    if (this.state.err) {
      return (
        <div className="page">
          <h2 style={{ color: "#b00" }}>Something broke.</h2>
          <pre style={{ whiteSpace: "pre-wrap", fontSize: 13 }}>
            {this.state.err.stack ?? this.state.err.message}
          </pre>
          <p className="muted">Check the console, then reload.</p>
        </div>
      );
    }
    return this.props.children;
  }
}

const root = document.getElementById("root");
if (!root) throw new Error("#root not found");
createRoot(root).render(
  <StrictMode>
    <ErrorBoundary>
      <App />
    </ErrorBoundary>
  </StrictMode>,
);
