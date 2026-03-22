import { createBrowserRouter } from "react-router";
import { Home } from "./pages/Home";
import { Result } from "./pages/Result";
import { History } from "./pages/History";
import { Auth } from "./pages/Auth";
import { Dashboard } from "./pages/Dashboard";
import { Layout } from "./components/Layout";

export const router = createBrowserRouter([
  {
    path: "/",
    Component: Layout,
    children: [
      { index: true, Component: Home },
      { path: "dashboard", Component: Dashboard },
      { path: "result/:id", Component: Result },
      { path: "history", Component: History },
      { path: "auth", Component: Auth },
    ],
  },
]);