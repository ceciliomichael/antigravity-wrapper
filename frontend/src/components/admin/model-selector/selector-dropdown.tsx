"use client";

import * as React from "react";
import { Check, Search } from "lucide-react";
import { ModelInfo } from "../../../types/admin";
import { Portal } from "../../ui/portal";

interface DropdownPosition {
  top: number;
  left: number;
  width: number;
}

interface SelectorDropdownProps {
  models: ModelInfo[];
  selectedModels: string[];
  searchQuery: string;
  position: DropdownPosition;
  dropdownId: string;
  onSearchChange: (query: string) => void;
  onToggle: (modelId: string) => void;
  onSelectAll: () => void;
  onClearAll: () => void;
}

export function SelectorDropdown({
  models,
  selectedModels,
  searchQuery,
  position,
  dropdownId,
  onSearchChange,
  onToggle,
  onSelectAll,
  onClearAll,
}: SelectorDropdownProps) {
  const inputRef = React.useRef<HTMLInputElement>(null);
  const [windowWidth, setWindowWidth] = React.useState(0);

  React.useEffect(() => {
    inputRef.current?.focus();
    setWindowWidth(window.innerWidth);
    
    const handleResize = () => setWindowWidth(window.innerWidth);
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  const filteredModels = React.useMemo(() => {
    if (!searchQuery.trim()) return models;
    const query = searchQuery.toLowerCase();
    return models.filter(
      (model) =>
        model.display_name.toLowerCase().includes(query) ||
        model.id.toLowerCase().includes(query)
    );
  }, [models, searchQuery]);

  const allSelected = selectedModels.length === models.length && models.length > 0;

  // Calculate safe positioning to prevent horizontal overflow
  const safeWidth = windowWidth > 0 ? Math.min(position.width, windowWidth - 32) : position.width;
  const safeLeft = windowWidth > 0 ? Math.min(position.left, windowWidth - safeWidth - 16) : position.left;

  return (
    <Portal>
      <div
        data-dropdown-id={dropdownId}
        className="fixed z-[9999] rounded-xl border border-zinc-200 bg-white shadow-xl overflow-hidden"
        style={{
          top: position.top,
          left: safeLeft,
          width: safeWidth,
          maxWidth: 'calc(100vw - 2rem)',
        }}
      >
        {/* Search Header */}
        <div className="border-b border-zinc-100 p-3">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
            <input
              ref={inputRef}
              type="text"
              value={searchQuery}
              onChange={(e) => onSearchChange(e.target.value)}
              placeholder="Search models..."
              className="h-10 w-full rounded-lg border border-zinc-200 bg-zinc-50 pl-10 pr-4 text-sm text-zinc-900 placeholder:text-zinc-400 transition-colors hover:border-zinc-300"
            />
          </div>
        </div>

        {/* Quick Actions */}
        <div className="flex items-center gap-2 border-b border-zinc-100 px-3 py-2 bg-zinc-50/50">
          <button
            type="button"
            onClick={onSelectAll}
            disabled={allSelected}
            className="text-xs font-medium text-zinc-600 hover:text-zinc-900 disabled:text-zinc-300 transition-colors"
          >
            Select All
          </button>
          <span className="text-zinc-300">|</span>
          <button
            type="button"
            onClick={onClearAll}
            disabled={selectedModels.length === 0}
            className="text-xs font-medium text-zinc-600 hover:text-zinc-900 disabled:text-zinc-300 transition-colors"
          >
            Clear All
          </button>
          <span className="ml-auto text-xs text-zinc-400">
            {selectedModels.length} of {models.length} selected
          </span>
        </div>

        {/* Model List */}
        <div className="max-h-64 overflow-y-auto p-2">
          {filteredModels.length === 0 ? (
            <div className="px-3 py-8 text-center text-sm text-zinc-400">
              {searchQuery ? "No models match your search" : "No models available"}
            </div>
          ) : (
            <div className="space-y-1">
              {filteredModels.map((model) => {
                const isSelected = selectedModels.includes(model.id);
                return (
                  <button
                    key={model.id}
                    type="button"
                    onClick={() => onToggle(model.id)}
                    className={`
                      flex w-full items-center justify-between rounded-lg px-3 py-2.5
                      text-left transition-all duration-100
                      ${isSelected 
                        ? "bg-zinc-100 hover:bg-zinc-150" 
                        : "hover:bg-zinc-50"
                      }
                    `}
                  >
                    <div className="flex flex-col min-w-0 flex-1">
                      <span className="text-sm font-medium text-zinc-900 truncate">
                        {model.display_name}
                      </span>
                      <span className="text-xs text-zinc-400 truncate">
                        {model.id}
                      </span>
                    </div>
                    <div
                      className={`
                        ml-3 flex h-5 w-5 shrink-0 items-center justify-center rounded-md border transition-all
                        ${isSelected
                          ? "border-zinc-900 bg-zinc-900 text-white"
                          : "border-zinc-300 bg-white"
                        }
                      `}
                    >
                      {isSelected && <Check className="h-3 w-3" />}
                    </div>
                  </button>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </Portal>
  );
}