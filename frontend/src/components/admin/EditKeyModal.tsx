"use client";

import * as React from "react";
import { ApiKey, ModelInfo } from "../../types/admin";
import { Modal, ModalHeader, ModalBody, ModalFooter } from "../ui/modal";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { ModelSelector } from "./model-selector";
import { Copy, Eye, EyeOff, Check } from "lucide-react";

interface EditKeyModalProps {
  apiKey: ApiKey | null;
  models: ModelInfo[];
  isOpen: boolean;
  onClose: () => void;
  onSave: (key: string, note: string, rateLimit: number, allowedModels: string[]) => Promise<void>;
}

export function EditKeyModal({
  apiKey,
  models,
  isOpen,
  onClose,
  onSave,
}: EditKeyModalProps) {
  const [loading, setLoading] = React.useState(false);
  const [isKeyVisible, setIsKeyVisible] = React.useState(false);
  const [copied, setCopied] = React.useState(false);
  const [form, setForm] = React.useState({
    note: "",
    rate_limit: "",
    allowed_models: [] as string[],
  });

  // Reset form when apiKey changes
  React.useEffect(() => {
    if (apiKey) {
      setForm({
        note: apiKey.note || "",
        rate_limit: apiKey.rate_limit ? apiKey.rate_limit.toString() : "",
        allowed_models: apiKey.allowed_models || [],
      });
      setIsKeyVisible(false);
    }
  }, [apiKey]);

  const handleSave = async () => {
    if (!apiKey) return;
    
    setLoading(true);
    try {
      await onSave(
        apiKey.key,
        form.note,
        parseInt(form.rate_limit) || 0,
        form.allowed_models
      );
      onClose();
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    if (!loading) {
      onClose();
    }
  };

  const copyToClipboard = () => {
    if (apiKey) {
      navigator.clipboard.writeText(apiKey.key);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  if (!apiKey) return null;

  return (
    <Modal isOpen={isOpen} onClose={handleClose} className="max-w-xl">
      <ModalHeader onClose={handleClose}>Edit API Key</ModalHeader>
      
      <ModalBody className="space-y-5">
        {/* API Key Display */}
        <div>
          <label className="mb-2 block text-sm font-medium text-zinc-700">
            API Key
          </label>
          <div className="flex items-center gap-2">
            <div className="relative flex-1">
              <Input
                type={isKeyVisible ? "text" : "password"}
                value={apiKey.key}
                readOnly
                className="h-11 pr-10 font-mono text-sm bg-zinc-50 border-zinc-200"
              />
              <button
                type="button"
                onClick={() => setIsKeyVisible(!isKeyVisible)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600 transition-colors"
              >
                {isKeyVisible ? <EyeOff size={16} /> : <Eye size={16} />}
              </button>
            </div>
            <Button
              type="button"
              variant="outline"
              size="icon"
              onClick={copyToClipboard}
              className={`h-11 w-11 shrink-0 transition-colors ${
                copied ? "border-emerald-300 text-emerald-600" : ""
              }`}
            >
              {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
            </Button>
          </div>
          <p className="mt-1.5 text-xs text-zinc-400">
            Created on {new Date(apiKey.created_at).toLocaleDateString()}
          </p>
        </div>

        {/* Note */}
        <div>
          <label className="mb-2 block text-sm font-medium text-zinc-700">
            Note
            <span className="ml-1 text-zinc-400 font-normal">(optional)</span>
          </label>
          <Input
            value={form.note}
            onChange={(e) => setForm({ ...form, note: e.target.value })}
            placeholder="e.g. Production App, Development Key"
            className="h-11"
          />
        </div>

        {/* Rate Limit */}
        <div>
          <label className="mb-2 block text-sm font-medium text-zinc-700">
            Rate Limit (RPM)
          </label>
          <Input
            type="number"
            value={form.rate_limit}
            onChange={(e) => setForm({ ...form, rate_limit: e.target.value })}
            placeholder="0 = unlimited"
            className="h-11 w-40"
          />
          <p className="mt-1.5 text-xs text-zinc-400">
            Maximum requests per minute. Leave empty or 0 for unlimited.
          </p>
        </div>

        {/* Allowed Models */}
        <div>
          <label className="mb-2 block text-sm font-medium text-zinc-700">
            Allowed Models
          </label>
          <ModelSelector
            models={models}
            selectedModels={form.allowed_models}
            onChange={(m) => setForm({ ...form, allowed_models: m })}
            showTags={true}
          />
          <p className="mt-2 text-xs text-zinc-400">
            Leave empty to allow access to all models
          </p>
        </div>
      </ModalBody>

      <ModalFooter>
        <Button
          type="button"
          variant="outline"
          onClick={handleClose}
          disabled={loading}
          className="h-10"
        >
          Cancel
        </Button>
        <Button
          type="button"
          onClick={handleSave}
          loading={loading}
          className="h-10"
        >
          Save Changes
        </Button>
      </ModalFooter>
    </Modal>
  );
}