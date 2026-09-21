import React from "react"
import ReactDOM from "react-dom/client"

import App from "./App"
import { Toaster } from "./components/ui/toast"
import { TooltipProvider } from "./components/ui/tooltip"
import "./index.css"

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <Toaster>
      <TooltipProvider>
        <App />
      </TooltipProvider>
    </Toaster>
  </React.StrictMode>,
)
