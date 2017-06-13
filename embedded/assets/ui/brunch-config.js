module.exports = {
  files: {
    javascripts: {
      joinTo: 'build.js'
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
