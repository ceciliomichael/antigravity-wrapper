"use client";

import * as React from "react";
import { useAdmin } from "../../hooks/use-admin";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../ui/card";
import { ShieldCheck } from "lucide-react";

export function LoginGate({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, login, isLoading, error } = useAdmin();
  const [inputSecret, setInputSecret] = React.useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await login(inputSecret);
    } catch {
      // Error is handled by context state
    }
  };

  if (isAuthenticated) {
    return <>{children}</>;
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-50 p-4">
      <Card className="w-full max-w-md border-zinc-200 bg-white">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-zinc-100">
            <ShieldCheck className="h-6 w-6 text-zinc-900" />
          </div>
          <CardTitle className="text-xl text-zinc-900">Admin Authentication</CardTitle>
          <CardDescription className="text-zinc-500">
            Enter your master secret to continue
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <Input
              type="password"
              placeholder="Master Secret"
              value={inputSecret}
              onChange={(e) => setInputSecret(e.target.value)}
              className="bg-white border-zinc-200 text-zinc-900 placeholder:text-zinc-400 focus:ring-zinc-200"
            />
            {error && (
              <div className="rounded-md bg-red-50 p-3 text-sm text-red-600 border border-red-200">
                {error}
              </div>
            )}
            <Button
              type="submit"
              className="w-full"
              loading={isLoading}
            >
              Unlock Dashboard
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}