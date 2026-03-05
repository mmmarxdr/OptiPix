import React, { useRef, useState } from 'react';

export default function DropZone({ onFiles, disabled }) {
    const [isDragActive, setIsDragActive] = useState(false);
    const dragCounter = useRef(0);
    const inputRef = useRef(null);

    const handleDragEnter = (e) => {
        e.preventDefault();
        e.stopPropagation();
        dragCounter.current += 1;
        if (e.dataTransfer.items && e.dataTransfer.items.length > 0) {
            setIsDragActive(true);
        }
    };

    const handleDragLeave = (e) => {
        e.preventDefault();
        e.stopPropagation();
        dragCounter.current -= 1;
        if (dragCounter.current === 0) {
            setIsDragActive(false);
        }
    };

    const handleDragOver = (e) => {
        e.preventDefault();
        e.stopPropagation();
    };

    const handleDrop = (e) => {
        e.preventDefault();
        e.stopPropagation();
        setIsDragActive(false);
        dragCounter.current = 0;
        if (disabled) return;
        if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
            onFiles(e.dataTransfer.files);
            e.dataTransfer.clearData();
        }
    };

    const handleChange = (e) => {
        if (disabled) return;
        if (e.target.files && e.target.files.length > 0) {
            onFiles(e.target.files);
            e.target.value = null; // reset
        }
    };

    const openFileDialog = () => {
        if (disabled) return;
        inputRef.current?.click();
    };

    const handleKeyDown = (e) => {
        if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            openFileDialog();
        }
    }

    return (
        <div
            className={`dropzone ${isDragActive ? 'dropzone--active' : ''} ${disabled ? 'dropzone--disabled' : ''}`}
            onDragEnter={handleDragEnter}
            onDragLeave={handleDragLeave}
            onDragOver={handleDragOver}
            onDrop={handleDrop}
            onClick={openFileDialog}
            onKeyDown={handleKeyDown}
            role="button"
            tabIndex={disabled ? -1 : 0}
        >
            <input
                type="file"
                ref={inputRef}
                onChange={handleChange}
                multiple
                accept=".jpg,.jpeg,.png,.webp,.gif,.tiff,.bmp,.avif,.heif,.svg"
                style={{ display: 'none' }}
            />
            <div className="dropzone__icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <polyline points="17 8 12 3 7 8" />
                    <line x1="12" y1="3" x2="12" y2="15" />
                </svg>
            </div>
            <div className="dropzone__title">
                {isDragActive ? 'Drop files here' : 'Drag & drop files here'}
            </div>
            <div className="dropzone__subtitle">
                Supports JPEG, PNG, WebP, SVG, AVIF and more
            </div>
        </div>
    );
}
