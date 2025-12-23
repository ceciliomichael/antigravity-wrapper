"use client";

import * as React from "react";
import { AdminClient } from "../lib/api";
import { ApiKey, GenerateKeyRequest, UpdateKeyRequest, ModelInfo } from "../types/admin";

interface AdminContextType {
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  keys: ApiKey[];
  models: ModelInfo[];
  login: (secret: string) => Promise<void>;
  logout: () => void;
  refreshKeys: () => Promise<void>;
  refreshModels: () => Promise<void>;
  createKey: (req: GenerateKeyRequest) => Promise<void>;
  updateKey: (key: string, req: UpdateKeyRequest) => Promise<void>;
  revokeKey: (key: string) => Promise<void>;
}

const AdminContext = React.createContext<AdminContextType | undefined>(
  undefined
);

export function AdminProvider({ children }: { children: React.ReactNode }) {
  const [secret, setSecret] = React.useState<string | null>(null);
  const [keys, setKeys] = React.useState<ApiKey[]>([]);
  const [models, setModels] = React.useState<ModelInfo[]>([]);
  const [isLoading, setIsLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  // Initialize from localStorage on mount
  React.useEffect(() => {
    const stored = localStorage.getItem("admin_secret");
    if (stored) {
      setSecret(stored);
    }
  }, []);

  // Create client instance when secret changes
  const client = React.useMemo(
    () => (secret ? new AdminClient(secret) : null),
    [secret]
  );

  const refreshKeys = React.useCallback(async () => {
    if (!client) return;
    setIsLoading(true);
    setError(null);
    try {
      const res = await client.listKeys();
      // Sort keys by creation date (newest first)
      const sorted = res.data.sort(
        (a, b) =>
          new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
      );
      setKeys(sorted);
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : "Unknown error";
      setError(errorMessage);
      if (errorMessage.includes("401") || errorMessage.includes("Unauthorized")) {
        logout(); // Auto logout on auth error
      }
    } finally {
      setIsLoading(false);
    }
  }, [client]);

  const refreshModels = React.useCallback(async () => {
    if (!client) return;
    try {
      const res = await client.listModels();
      setModels(res.data);
    } catch (err: unknown) {
      // Silently fail for models - not critical
      console.error("Failed to fetch models:", err);
    }
  }, [client]);

  // Initial fetch when secret is set
  React.useEffect(() => {
    if (secret) {
      refreshKeys();
      refreshModels();
    }
  }, [secret, refreshKeys, refreshModels]);

  const login = async (newSecret: string) => {
    setIsLoading(true);
    setError(null);
    try {
      const testClient = new AdminClient(newSecret);
      await testClient.listKeys(); // Verify secret works
      setSecret(newSecret);
      localStorage.setItem("admin_secret", newSecret);
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : "Unknown error";
      setError(errorMessage);
      throw err;
    } finally {
      setIsLoading(false);
    }
  };

  const logout = () => {
    setSecret(null);
    setKeys([]);
    setModels([]);
    localStorage.removeItem("admin_secret");
  };

  const createKey = async (req: GenerateKeyRequest) => {
    if (!client) return;
    setIsLoading(true);
    setError(null);
    try {
      await client.generateKey(req);
      await refreshKeys();
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : "Unknown error";
      setError(errorMessage);
      throw err;
    } finally {
      setIsLoading(false);
    }
  };

  const updateKey = async (key: string, req: UpdateKeyRequest) => {
    if (!client) return;
    setIsLoading(true);
    setError(null);
    try {
      await client.updateKey(key, req);
      await refreshKeys();
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : "Unknown error";
      setError(errorMessage);
      throw err;
    } finally {
      setIsLoading(false);
    }
  };

  const revokeKey = async (key: string) => {
    if (!client) return;
    setIsLoading(true);
    setError(null);
    try {
      await client.revokeKey(key);
      await refreshKeys();
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : "Unknown error";
      setError(errorMessage);
      throw err;
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <AdminContext.Provider
      value={{
        isAuthenticated: !!secret,
        isLoading,
        error,
        keys,
        models,
        login,
        logout,
        refreshKeys,
        refreshModels,
        createKey,
        updateKey,
        revokeKey,
      }}
    >
      {children}
    </AdminContext.Provider>
  );
}

export function useAdmin() {
  const context = React.useContext(AdminContext);
  if (context === undefined) {
    throw new Error("useAdmin must be used within an AdminProvider");
  }
  return context;
}