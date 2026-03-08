import React, { useState } from 'react';
import { useOptimizer } from './hooks/useOptimizer';
import DropZone from './components/DropZone';
import OptionsPanel from './components/OptionsPanel';
import FileList from './components/FileList';
import optipixLogo from './assets/optipix-logo.svg';
import './styles.css';

function App() {
    const [options, setOptions] = useState({
        format: 'webp',
        quality: 80,
        width: 0,
        height: 0,
        stripMetadata: true,
        lossless: false,
        effort: 4
    });

    const {
        files,
        results,
        processing,
        addFiles,
        removeFile,
        clearAll,
        processAll,
        downloadResult,
        downloadAll
    } = useOptimizer();

    return (
        <div className="app">
            <div className="bg-grain"></div>
            <div className="bg-glow--1"></div>
            <div className="bg-glow--2"></div>

            <header className="header">
                <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
                    <div className="header__logo">
                        <img src={optipixLogo} alt="OptiPix Logo" width="24" height="24" />
                    </div>
                    <div>
                        <h1 style={{ margin: 0, fontSize: '1.5rem', fontWeight: 700, color: 'var(--text)' }}>OptiPix</h1>
                        <div style={{ fontSize: '0.85rem', color: 'var(--text-muted)' }}>High-performance media optimizer</div>
                    </div>
                </div>
                <div style={{ display: 'flex', gap: '8px' }}>
                    <span className="badge badge--accent">libvips</span>
                    <span className="badge">WebP</span>
                    <span className="badge">AVIF</span>
                    <span className="badge">SVG</span>
                </div>
            </header>

            <main className="main">
                <div className="main__left">
                    <DropZone onFiles={addFiles} disabled={processing} />
                    <FileList
                        files={files}
                        results={results}
                        onRemove={removeFile}
                        onDownload={downloadResult}
                        onDownloadAll={downloadAll}
                        onClear={clearAll}
                    />

                    <div className="info-grid">
                        <div className="info-card">
                            <h4 style={{ margin: '0 0 12px 0' }}>How it works</h4>
                            <ol style={{ margin: 0, paddingLeft: '20px', fontSize: '0.9rem', color: 'var(--text-muted)' }}>
                                <li style={{ marginBottom: '8px' }}>Drop your images or SVGs</li>
                                <li style={{ marginBottom: '8px' }}>Choose output format and quality</li>
                                <li style={{ marginBottom: '8px' }}>Images are processed blazingly fast by libvips in Go</li>
                                <li>Download your optimized files</li>
                            </ol>
                        </div>

                        <div className="info-card info-card--tech">
                            <h4 style={{ margin: '0 0 8px 0', display: 'flex', alignItems: 'center', gap: '8px' }}>
                                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                    <circle cx="12" cy="12" r="10"></circle>
                                    <line x1="12" y1="16" x2="12" y2="12"></line>
                                    <line x1="12" y1="8" x2="12.01" y2="8"></line>
                                </svg>
                                Engine
                            </h4>
                            <p style={{ margin: 0, fontSize: '0.85rem', lineHeight: 1.5 }}>
                                Powered by <strong>govips</strong> (Go bindings for libvips) for raster images and <strong>SVGO</strong> for vector graphics. Processing is stateless and files are not stored on the server.
                            </p>
                        </div>
                    </div>
                </div>

                <div className="main__right">
                    <OptionsPanel
                        options={options}
                        onChange={setOptions}
                        fileCount={files.length}
                        onProcess={() => processAll(options)}
                        processing={processing}
                    />
                </div>
            </main>

            <footer className="footer">
                OptiPix v1.0 · Go + libvips + SVGO + React
            </footer>
        </div>
    );
}

export default App;
