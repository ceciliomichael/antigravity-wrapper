"use client";

import * as React from "react";
import { X } from "lucide-react";
import { ModelInfo } from "../../../types/admin";

interface TagsDisplayProps {
  models: ModelInfo[];
  selectedModels: string[];
  onRemove: (modelId: string) => void;
  disabled?: boolean;
  maxVisible?: number;
}

export function TagsDisplay({
  models,
  selectedModels,
  onRemove,
  disabled = false,
  maxVisible = 6,
}: TagsDisplayProps) {
  const [showAll, setShowAll] = React.useState(false);

  if (selectedModels.length === 0) {
    return null;
  }

  const getModelDisplayName = (modelId: string) => {
    const model = models.find((m) => m.id === modelId);
    return model?.display_name || modelId;
  };

  const visibleModels = showAll 
    ? selectedModels 
    : selectedModels.slice(0, maxVisible);
  
  const hiddenCount = selectedModels.length - maxVisible;

  return (
    <div className="mt-3 rounded-lg border border-zinc-100 bg-zinc-50/50 p-3">
      <div className="flex flex-wrap gap-2">
        {visibleModels.map((modelId) => (
          <span
            key={modelId}
            className="inline-flex items-center gap-1.5 rounded-lg border border-zinc-200 bg-white px-3 py-1.5 text-sm text-zinc-700 shadow-sm"
          >
            <span className="max-w-[180px] truncate">
              {getModelDisplayName(modelId)}
            </span>
            {!disabled && (
              <button
                type="button"
                onClick={() => onRemove(modelId)}
                className="rounded-full p-0.5 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-600 transition-colors"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            )}
          </span>
        ))}
        
        {hiddenCount > 0 && !showAll && (
          <button
            type="button"
            onClick={() => setShowAll(true)}
            className="inline-flex items-center rounded-lg border border-zinc-200 bg-white px-3 py-1.5 text-sm font-medium text-zinc-500 hover:bg-zinc-50 hover:text-zinc-700 transition-colors"
          >
            +{hiddenCount} more
          </button>
        )}
        
        {showAll && hiddenCount > 0 && (
          <button
            type="button"
            onClick={() => setShowAll(false)}
            className="inline-flex items-center rounded-lg border border-zinc-200 bg-white px-3 py-1.5 text-sm font-medium text-zinc-500 hover:bg-zinc-50 hover:text-zinc-700 transition-colors"
          >
            Show less
          </button>
        )}
      </div>
    </div>
  );
}