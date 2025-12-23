"use client";

import { useAdmin } from "../../hooks/use-admin";
import { CreateKeyForm } from "./CreateKeyForm";
import { KeyList } from "./KeyList";
import { Button } from "../ui/button";
import { LogOut, Key } from "lucide-react";

export function Dashboard() {
  const { logout } = useAdmin();

  return (
    <div className="min-h-screen bg-zinc-50 overflow-x-hidden">
      {/* Header */}
      <header className="sticky top-0 z-40 border-b border-zinc-200 bg-white/80 backdrop-blur-sm">
        <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-12">
          <div className="flex h-16 items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-zinc-900">
                <Key className="h-4 w-4 text-white" />
              </div>
              <div>
                <h1 className="text-lg font-semibold text-zinc-900">Antigravity Wrapper</h1>
                <p className="text-xs text-zinc-500">Admin Dashboard</p>
              </div>
            </div>
            <Button 
              variant="ghost" 
              onClick={logout} 
              className="h-9 gap-2 text-zinc-600 hover:bg-zinc-100 hover:text-zinc-900"
            >
              <LogOut className="h-4 w-4" />
              <span className="hidden sm:inline">Logout</span>
            </Button>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="mx-auto max-w-6xl px-4 py-6 sm:px-6 sm:py-10 lg:px-12 lg:py-16">
        <div className="space-y-8 lg:space-y-12">
          {/* Page Title */}
          <div>
            <h2 className="text-2xl font-bold tracking-tight text-zinc-900 sm:text-3xl">
              API Key Management
            </h2>
            <p className="mt-2 text-sm text-zinc-500 sm:text-base">
              Create and manage API keys for accessing the Antigravity Wrapper services
            </p>
          </div>

          {/* Cards */}
          <div className="grid gap-8 lg:gap-10">
            <CreateKeyForm />
            <KeyList />
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t border-zinc-200 bg-white">
        <div className="mx-auto max-w-6xl px-4 py-6 sm:px-6 lg:px-12">
          <p className="text-center text-xs text-zinc-400">
            Antigravity Wrapper Admin Panel
          </p>
        </div>
      </footer>
    </div>
  );
}