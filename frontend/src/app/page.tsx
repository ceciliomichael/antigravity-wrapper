import { AdminProvider } from "../hooks/use-admin";
import { LoginGate } from "../components/admin/LoginGate";
import { Dashboard } from "../components/admin/Dashboard";

export default function Home() {
  return (
    <AdminProvider>
      <LoginGate>
        <Dashboard />
      </LoginGate>
    </AdminProvider>
  );
}