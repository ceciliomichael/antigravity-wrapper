"use client";

import * as React from "react";
import { ChevronDown, X } from "lucide-react";

interface SelectorTriggerProps {
  selectedCount: number;
  isOpen: boolean;
  disabled?: boolean;
  placeholder?: string;
  onClick: () => void;
  onClear: () => void;
}

export function SelectorTrigger({
  selectedCount,
  isOpen,
  disabled = false,
  placeholder = "All Models (unrestricted)",
  onClick,
  onClear,
}: SelectorTriggerProps) {
  const displayText = selectedCount === 0 
    ? placeholder 
    : `${selectedCount} model${selectedCount > 1 ? "s" : ""} selected`;

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={`
        flex h-11 w-full items-center justify-between
        rounded-lg border border-zinc-200 bg-white px-4
        text-left text-sm transition-all duration-150
        ${disabled 
          ? "cursor-not-allowed opacity-50" 
          : "hover:border-zinc-300 hover:shadow-sm"
        }
        ${isOpen ? "border-zinc-400 ring-2 ring-zinc-100" : ""}
      `}
    >
      <span className={selectedCount === 0 ? "text-zinc-400" : "text-zinc-700 font-medium"}>
        {displayText}
      </span>
      
      <div className="flex items-center gap-2">
        {selectedCount > 0 && !disabled && (
          <span
            role="button"
            tabIndex={0}
            onClick={(e) => {
              e.stopPropagation();
              onClear();
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.stopPropagation();
                onClear();
              }
            }}
            className="rounded-full p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-600 transition-colors"
          >
            <X className="h-4 w-4" />
          </span>
        )}
        <ChevronDown
          className={`h-4 w-4 text-zinc-400 transition-transform duration-200 ${
            isOpen ? "rotate-180" : ""
          }`}
        />
      </div>
    </button>
  );
}