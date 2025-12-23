"use client";

import * as React from "react";
import { ModelInfo } from "../../../types/admin";
import { SelectorTrigger } from "./selector-trigger";
import { SelectorDropdown } from "./selector-dropdown";
import { TagsDisplay } from "./tags-display";

interface DropdownPosition {
  top: number;
  left: number;
  width: number;
}

interface ModelSelectorProps {
  models: ModelInfo[];
  selectedModels: string[];
  onChange: (models: string[]) => void;
  disabled?: boolean;
  showTags?: boolean;
  compact?: boolean;
}

export function ModelSelector({
  models,
  selectedModels,
  onChange,
  disabled = false,
  showTags = true,
  compact = false,
}: ModelSelectorProps) {
  const [isOpen, setIsOpen] = React.useState(false);
  const [searchQuery, setSearchQuery] = React.useState("");
  const [position, setPosition] = React.useState<DropdownPosition>({ top: 0, left: 0, width: 0 });
  const triggerRef = React.useRef<HTMLDivElement>(null);
  const dropdownId = React.useId();

  // Calculate dropdown position when opening
  const updatePosition = React.useCallback(() => {
    if (triggerRef.current) {
      const rect = triggerRef.current.getBoundingClientRect();
      setPosition({
        top: rect.bottom + 8,
        left: rect.left,
        width: rect.width,
      });
    }
  }, []);

  // Update position on scroll/resize when open
  React.useEffect(() => {
    if (!isOpen) return;

    updatePosition();

    const handleScrollOrResize = () => {
      updatePosition();
    };

    window.addEventListener("scroll", handleScrollOrResize, true);
    window.addEventListener("resize", handleScrollOrResize);

    return () => {
      window.removeEventListener("scroll", handleScrollOrResize, true);
      window.removeEventListener("resize", handleScrollOrResize);
    };
  }, [isOpen, updatePosition]);

  // Close dropdown when clicking outside
  React.useEffect(() => {
    if (!isOpen) return;

    function handleClickOutside(event: MouseEvent) {
      const target = event.target as Node;
      const triggerElement = triggerRef.current;
      const dropdownElement = document.querySelector(`[data-dropdown-id="${dropdownId}"]`);

      const clickedTrigger = triggerElement?.contains(target);
      const clickedDropdown = dropdownElement?.contains(target);

      if (!clickedTrigger && !clickedDropdown) {
        setIsOpen(false);
        setSearchQuery("");
      }
    }

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [isOpen, dropdownId]);

  // Close on escape
  React.useEffect(() => {
    if (!isOpen) return;

    function handleEscape(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setIsOpen(false);
        setSearchQuery("");
      }
    }

    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [isOpen]);

  const toggleModel = (modelId: string) => {
    if (selectedModels.includes(modelId)) {
      onChange(selectedModels.filter((id) => id !== modelId));
    } else {
      onChange([...selectedModels, modelId]);
    }
  };

  const removeModel = (modelId: string) => {
    onChange(selectedModels.filter((id) => id !== modelId));
  };

  const selectAll = () => {
    onChange(models.map((m) => m.id));
  };

  const clearAll = () => {
    onChange([]);
  };

  const handleTriggerClick = () => {
    if (!disabled) {
      if (!isOpen) {
        updatePosition();
      }
      setIsOpen(!isOpen);
      if (isOpen) {
        setSearchQuery("");
      }
    }
  };

  return (
    <div className="relative w-full">
      <div ref={triggerRef}>
        <SelectorTrigger
          selectedCount={selectedModels.length}
          isOpen={isOpen}
          disabled={disabled}
          onClick={handleTriggerClick}
          onClear={clearAll}
        />
      </div>

      {isOpen && !disabled && (
        <SelectorDropdown
          models={models}
          selectedModels={selectedModels}
          searchQuery={searchQuery}
          position={position}
          dropdownId={dropdownId}
          onSearchChange={setSearchQuery}
          onToggle={toggleModel}
          onSelectAll={selectAll}
          onClearAll={clearAll}
        />
      )}

      {showTags && !compact && (
        <TagsDisplay
          models={models}
          selectedModels={selectedModels}
          onRemove={removeModel}
          disabled={disabled}
        />
      )}
    </div>
  );
}