module.exports = {
  npm: {
    enabled: true,
    static: [
      'node_modules/react/dist/react.min.js',
      'node_modules/react-dom/dist/react-dom.min.js',
    ]
  },
  files: {
    javascripts: {
      joinTo: {
        'build.js': /^app/,
        'vendor.js': /^(node_modules)/,
      },
      order: {
        before: [
          'node_modules/react/dist/react.min.js',
          'node_modules/react-dom/dist/react-dom.min.js'
        ]
      }
    },
    stylesheets: {
      joinTo: 'build.css',
    }
  },
  modules: {
    allSourceFiles: false,
    autoRequire: {
      'render.jsx': ['render.jsx']
    }
  },
  plugins: {
    postcss: {processors: [require('autoprefixer')]},
    babel: {
      presets: ['latest', 'react'],
      pattern: /^app\/.*\.jsx/
    },
    uglify: {
      mangle: true,
      compress: {
        global_defs: {
          DEBUG: false
        }
      }
    },
  }
};
