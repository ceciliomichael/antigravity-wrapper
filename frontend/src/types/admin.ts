export interface ApiKey {
  key: string;
  created_at: string;
  note?: string;
  rate_limit?: number;
  allowed_models?: string[];
}

export interface GenerateKeyRequest {
  note: string;
  rate_limit: number;
  allowed_models?: string[];
}

export interface UpdateKeyRequest {
  note: string;
  rate_limit: number;
  allowed_models?: string[];
}

export interface GenerateKeyResponse {
  key: string;
  created_at: string;
  note?: string;
  rate_limit?: number;
  allowed_models?: string[];
}

export interface ListKeysResponse {
  data: ApiKey[];
}

export interface ModelInfo {
  id: string;
  display_name: string;
}

export interface ListModelsResponse {
  data: ModelInfo[];
}

export interface ApiError {
  error: {
    message: string;
    type: string;
  };
}