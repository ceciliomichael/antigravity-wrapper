"use client";

import * as React from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";
import { cn } from "../../lib/utils";

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  children: React.ReactNode;
  className?: string;
}

interface ModalHeaderProps {
  children: React.ReactNode;
  onClose?: () => void;
  className?: string;
}

interface ModalBodyProps {
  children: React.ReactNode;
  className?: string;
}

interface ModalFooterProps {
  children: React.ReactNode;
  className?: string;
}

function ModalOverlay({ onClick }: { onClick: () => void }) {
  return (
    <div
      className="fixed inset-0 z-[9998] bg-black/50 backdrop-blur-sm animate-in fade-in duration-200"
      onClick={onClick}
      aria-hidden="true"
    />
  );
}

function ModalContent({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      role="dialog"
      aria-modal="true"
      className={cn(
        "fixed left-1/2 top-1/2 z-[9999] w-[calc(100%-2rem)] max-w-lg -translate-x-1/2 -translate-y-1/2",
        "rounded-xl border border-zinc-200 bg-white shadow-2xl",
        "animate-in fade-in zoom-in-95 duration-200",
        "max-h-[calc(100vh-2rem)] overflow-y-auto",
        "sm:w-full",
        className
      )}
      onClick={(e) => e.stopPropagation()}
    >
      {children}
    </div>
  );
}

export function Modal({ isOpen, onClose, children, className }: ModalProps) {
  const [mounted, setMounted] = React.useState(false);

  React.useEffect(() => {
    setMounted(true);
  }, []);

  React.useEffect(() => {
    if (!isOpen) return;

    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };

    document.addEventListener("keydown", handleEscape);
    document.body.style.overflow = "hidden";

    return () => {
      document.removeEventListener("keydown", handleEscape);
      document.body.style.overflow = "";
    };
  }, [isOpen, onClose]);

  if (!mounted || !isOpen) return null;

  return createPortal(
    <>
      <ModalOverlay onClick={onClose} />
      <ModalContent className={className}>{children}</ModalContent>
    </>,
    document.body
  );
}

export function ModalHeader({ children, onClose, className }: ModalHeaderProps) {
  return (
    <div
      className={cn(
        "flex items-center justify-between border-b border-zinc-100 px-6 py-4",
        className
      )}
    >
      <div className="text-lg font-semibold text-zinc-900">{children}</div>
      {onClose && (
        <button
          type="button"
          onClick={onClose}
          className="rounded-lg p-1.5 text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-600"
        >
          <X className="h-5 w-5" />
        </button>
      )}
    </div>
  );
}

export function ModalBody({ children, className }: ModalBodyProps) {
  return (
    <div className={cn("px-6 py-5", className)}>
      {children}
    </div>
  );
}

export function ModalFooter({ children, className }: ModalFooterProps) {
  return (
    <div
      className={cn(
        "flex items-center justify-end gap-3 border-t border-zinc-100 bg-zinc-50/50 px-6 py-4",
        className
      )}
    >
      {children}
    </div>
  );
}