<template>
  <div id="app">

    <header>
      <b-container fluid class="px-0">
        <div id="nav">
          <b-navbar toggleable="md" type="dark" variant="dark">
            <b-navbar-brand to="/">Enduro</b-navbar-brand>
            <b-navbar-toggle target="nav-collapse"></b-navbar-toggle>
            <b-collapse id="nav-collapse" is-nav>
              <b-navbar-nav>
                <b-nav-item :to="{ name: 'collections' }">Collections</b-nav-item>
              </b-navbar-nav>
              <b-navbar-nav class="ml-auto">
                <b-nav-item href="https://enduroproject.netlify.com">Docs</b-nav-item>
                <b-nav-item href="https://github.com/artefactual-labs/enduro">GitHub</b-nav-item>
              </b-navbar-nav>
            </b-collapse>
          </b-navbar>
        </div>
      </b-container>
    </header>

    <router-view/>

    <footer>
      <b-container navbar>
        Enduro is licensed under Apache License 2.0.
      </b-container>
    </footer>

  </div>
</template>

<style lang="scss">

@import 'node_modules/bootstrap/scss/bootstrap';
@import 'node_modules/bootstrap-vue/src/index.scss';

#app {
  font-family: 'Avenir', Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  color: #2c3e50;
}

#nav {
  margin-bottom: 10px;
  .nav-link.router-link-exact-active {
    color: white;
    background-color: #fff2;
    @extend .rounded-sm;
  }
}

.breadcrumb {
  background-color: #fff;
  padding: 0.2rem 1rem;
}

.bg-dark {
  background-color: #5e2750 !important;
}

footer {
  margin-top: 40px;
  .container {
    padding: 40px;
    text-align: center;
    border-top: 1px solid #eaecef;
    color: #999;
  }
}

</style>
