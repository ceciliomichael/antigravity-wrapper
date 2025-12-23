"use client";

import * as React from "react";
import { useAdmin } from "../../hooks/use-admin";
import { ApiKey, ModelInfo } from "../../types/admin";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../ui/table";
import { Badge } from "../ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../ui/card";
import { Copy, RefreshCw, Trash2, Pencil, Eye, EyeOff, Check } from "lucide-react";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { EditKeyModal } from "./EditKeyModal";

interface KeyItemProps {
  apiKey: ApiKey;
  models: ModelInfo[];
  onEdit: (key: ApiKey) => void;
  onRevoke: (key: string) => Promise<void>;
}

function useKeyActions(apiKey: ApiKey, onRevoke: (key: string) => Promise<void>) {
  const [isVisible, setIsVisible] = React.useState(false);
  const [loading, setLoading] = React.useState(false);
  const [copied, setCopied] = React.useState(false);

  const copyToClipboard = () => {
    navigator.clipboard.writeText(apiKey.key);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleRevoke = async () => {
    if (!confirm("Are you sure? This cannot be undone.")) return;
    setLoading(true);
    try {
      await onRevoke(apiKey.key);
    } finally {
      setLoading(false);
    }
  };

  return {
    isVisible,
    setIsVisible,
    loading,
    copied,
    copyToClipboard,
    handleRevoke,
  };
}

function getModelDisplayName(models: ModelInfo[], modelId: string) {
  const model = models.find((m) => m.id === modelId);
  return model?.display_name || modelId;
}

function ModelBadges({ apiKey, models }: { apiKey: ApiKey; models: ModelInfo[] }) {
  return (
    <div className="flex flex-wrap gap-1.5">
      {!apiKey.allowed_models || apiKey.allowed_models.length === 0 ? (
        <Badge 
          variant="secondary" 
          className="bg-emerald-50 text-emerald-700 border border-emerald-200 text-xs font-medium"
        >
          All Models
        </Badge>
      ) : (
        <>
          {apiKey.allowed_models.slice(0, 2).map((modelId) => (
            <Badge
              key={modelId}
              variant="secondary"
              className="bg-zinc-100 text-zinc-700 border border-zinc-200 text-xs"
            >
              {getModelDisplayName(models, modelId)}
            </Badge>
          ))}
          {apiKey.allowed_models.length > 2 && (
            <Badge 
              variant="secondary" 
              className="bg-zinc-50 text-zinc-500 border border-zinc-200 text-xs"
            >
              +{apiKey.allowed_models.length - 2} more
            </Badge>
          )}
        </>
      )}
    </div>
  );
}

function RateLimitBadge({ rateLimit }: { rateLimit?: number }) {
  return (
    <Badge 
      variant="secondary" 
      className={`
        border text-xs font-medium
        ${rateLimit 
          ? "bg-zinc-100 text-zinc-700 border-zinc-200" 
          : "bg-emerald-50 text-emerald-700 border-emerald-200"
        }
      `}
    >
      {rateLimit ? `${rateLimit} RPM` : "Unlimited"}
    </Badge>
  );
}

function KeyCard({ apiKey, models, onEdit, onRevoke }: KeyItemProps) {
  const { isVisible, setIsVisible, loading, copied, copyToClipboard, handleRevoke } = 
    useKeyActions(apiKey, onRevoke);

  return (
    <div className="rounded-xl border border-zinc-200 bg-white p-4 shadow-sm">
      {/* API Key Section */}
      <div className="mb-4">
        <div className="flex items-center gap-2 mb-2">
          <div className="relative flex-1">
            <Input
              type={isVisible ? "text" : "password"}
              value={apiKey.key}
              readOnly
              className="h-10 pr-10 font-mono text-xs bg-zinc-50 border-zinc-200"
            />
            <button
              type="button"
              onClick={() => setIsVisible(!isVisible)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600 transition-colors"
            >
              {isVisible ? <EyeOff size={14} /> : <Eye size={14} />}
            </button>
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={copyToClipboard}
            className={`h-10 w-10 shrink-0 transition-colors ${
              copied 
                ? "text-emerald-600 hover:text-emerald-600" 
                : "text-zinc-400 hover:text-zinc-600 hover:bg-zinc-100"
            }`}
          >
            {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
          </Button>
        </div>
        <p className="text-xs text-zinc-400">
          Created {new Date(apiKey.created_at).toLocaleDateString()}
        </p>
      </div>

      {/* Details Grid */}
      <div className="space-y-3 mb-4">
        {/* Note */}
        <div className="flex items-start justify-between gap-2">
          <span className="text-xs font-medium text-zinc-500 shrink-0">Note</span>
          <span className={`text-sm text-right ${apiKey.note ? "text-zinc-700" : "text-zinc-400 italic"}`}>
            {apiKey.note || "No note"}
          </span>
        </div>

        {/* Rate Limit */}
        <div className="flex items-center justify-between gap-2">
          <span className="text-xs font-medium text-zinc-500">Rate Limit</span>
          <RateLimitBadge rateLimit={apiKey.rate_limit} />
        </div>

        {/* Allowed Models */}
        <div>
          <span className="text-xs font-medium text-zinc-500 block mb-2">Allowed Models</span>
          <ModelBadges apiKey={apiKey} models={models} />
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center justify-end gap-2 pt-3 border-t border-zinc-100">
        <Button
          variant="outline"
          size="sm"
          onClick={() => onEdit(apiKey)}
          className="h-9 gap-1.5 text-zinc-600 hover:text-zinc-900"
        >
          <Pencil className="h-3.5 w-3.5" />
          Edit
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={handleRevoke}
          loading={loading}
          className="h-9 gap-1.5 text-red-500 hover:text-red-600 hover:bg-red-50 hover:border-red-200"
        >
          <Trash2 className="h-3.5 w-3.5" />
          Revoke
        </Button>
      </div>
    </div>
  );
}

function KeyRow({ apiKey, models, onEdit, onRevoke }: KeyItemProps) {
  const { isVisible, setIsVisible, loading, copied, copyToClipboard, handleRevoke } = 
    useKeyActions(apiKey, onRevoke);

  return (
    <TableRow className="border-zinc-200 hover:bg-zinc-50/50">
      {/* API Key Column */}
      <TableCell className="font-mono text-zinc-700">
        <div className="flex items-center gap-2">
          <div className="relative">
            <Input
              type={isVisible ? "text" : "password"}
              value={apiKey.key}
              readOnly
              className="h-9 w-[180px] pr-9 font-mono text-xs bg-zinc-50 border-zinc-200"
            />
            <button
              type="button"
              onClick={() => setIsVisible(!isVisible)}
              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600 transition-colors"
            >
              {isVisible ? <EyeOff size={14} /> : <Eye size={14} />}
            </button>
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={copyToClipboard}
            className={`h-9 w-9 transition-colors ${
              copied 
                ? "text-emerald-600 hover:text-emerald-600" 
                : "text-zinc-400 hover:text-zinc-600 hover:bg-zinc-100"
            }`}
          >
            {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
          </Button>
        </div>
        <div className="text-xs text-zinc-400 mt-1.5 pl-1">
          Created {new Date(apiKey.created_at).toLocaleDateString()}
        </div>
      </TableCell>

      {/* Note Column */}
      <TableCell className="text-zinc-700">
        <span className={apiKey.note ? "text-zinc-700" : "text-zinc-400 italic"}>
          {apiKey.note || "No note"}
        </span>
      </TableCell>

      {/* Rate Limit Column */}
      <TableCell>
        <RateLimitBadge rateLimit={apiKey.rate_limit} />
      </TableCell>

      {/* Allowed Models Column */}
      <TableCell>
        <ModelBadges apiKey={apiKey} models={models} />
      </TableCell>

      {/* Actions Column */}
      <TableCell className="text-right">
        <div className="flex items-center justify-end gap-1">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => onEdit(apiKey)}
            className="h-9 w-9 text-zinc-500 hover:text-zinc-900 hover:bg-zinc-100"
          >
            <Pencil className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={handleRevoke}
            loading={loading}
            className="h-9 w-9 text-red-500 hover:text-red-600 hover:bg-red-50"
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center gap-2 py-12 text-zinc-400">
      <div className="text-sm">No API keys found</div>
      <div className="text-xs">Create your first key above to get started</div>
    </div>
  );
}

export function KeyList() {
  const { keys, models, isLoading, refreshKeys, updateKey, revokeKey } = useAdmin();
  const [editingKey, setEditingKey] = React.useState<ApiKey | null>(null);

  const handleSaveKey = async (
    key: string,
    note: string,
    rateLimit: number,
    allowedModels: string[]
  ) => {
    await updateKey(key, {
      note,
      rate_limit: rateLimit,
      allowed_models: allowedModels,
    });
  };

  return (
    <>
      <Card className="border-zinc-200 bg-white shadow-sm">
        <CardHeader className="flex flex-row items-center justify-between pb-4">
          <div>
            <CardTitle className="text-lg sm:text-xl font-semibold text-zinc-900">Active Keys</CardTitle>
            <CardDescription className="text-zinc-500 text-sm">
              Manage your API keys, rate limits, and model access
            </CardDescription>
          </div>
          <Button
            variant="outline"
            size="icon"
            onClick={() => refreshKeys()}
            loading={isLoading}
            className="h-9 w-9 shrink-0 border-zinc-200 hover:bg-zinc-50"
          >
            <RefreshCw className="h-4 w-4" />
          </Button>
        </CardHeader>
        <CardContent className="px-4 sm:px-6">
          {keys.length === 0 ? (
            <EmptyState />
          ) : (
            <>
              {/* Mobile: Card Layout */}
              <div className="md:hidden space-y-4">
                {keys.map((key) => (
                  <KeyCard
                    key={key.key}
                    apiKey={key}
                    models={models}
                    onEdit={setEditingKey}
                    onRevoke={revokeKey}
                  />
                ))}
              </div>

              {/* Desktop: Table Layout */}
              <div className="hidden md:block">
                <Table>
                  <TableHeader>
                    <TableRow className="hover:bg-transparent border-zinc-200 bg-zinc-50/50">
                      <TableHead className="w-[260px] pl-4">API Key</TableHead>
                      <TableHead className="w-[180px]">Note</TableHead>
                      <TableHead className="w-[120px]">Rate Limit</TableHead>
                      <TableHead className="w-[200px]">Allowed Models</TableHead>
                      <TableHead className="text-right pr-4 w-[100px]">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {keys.map((key) => (
                      <KeyRow
                        key={key.key}
                        apiKey={key}
                        models={models}
                        onEdit={setEditingKey}
                        onRevoke={revokeKey}
                      />
                    ))}
                  </TableBody>
                </Table>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      <EditKeyModal
        apiKey={editingKey}
        models={models}
        isOpen={editingKey !== null}
        onClose={() => setEditingKey(null)}
        onSave={handleSaveKey}
      />
    </>
  );
}