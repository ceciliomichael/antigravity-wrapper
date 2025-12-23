import {
  GenerateKeyRequest,
  GenerateKeyResponse,
  ListKeysResponse,
  ListModelsResponse,
  UpdateKeyRequest,
  ApiKey,
  ApiError,
} from "../types/admin";

const API_BASE = ""; // Relative path since we're serving from the same domain or proxy

export class AdminClient {
  private secret: string;

  constructor(secret: string) {
    this.secret = secret;
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const headers = {
      "Content-Type": "application/json",
      Authorization: `Bearer ${this.secret}`,
      ...options.headers,
    };

    const response = await fetch(`${API_BASE}${endpoint}`, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const errorData = (await response.json().catch(() => ({}))) as ApiError;
      throw new Error(
        errorData.error?.message || `API Error: ${response.statusText}`
      );
    }

    return response.json();
  }

  async listKeys(): Promise<ListKeysResponse> {
    return this.request<ListKeysResponse>("/admin/keys");
  }

  async generateKey(req: GenerateKeyRequest): Promise<GenerateKeyResponse> {
    return this.request<GenerateKeyResponse>("/admin/keys", {
      method: "POST",
      body: JSON.stringify(req),
    });
  }

  async updateKey(key: string, req: UpdateKeyRequest): Promise<ApiKey> {
    return this.request<ApiKey>(`/admin/keys/${key}`, {
      method: "PUT",
      body: JSON.stringify(req),
    });
  }

  async revokeKey(key: string): Promise<void> {
    return this.request<void>(`/admin/keys/${key}`, {
      method: "DELETE",
    });
  }

  async listModels(): Promise<ListModelsResponse> {
    return this.request<ListModelsResponse>("/admin/models");
  }
}