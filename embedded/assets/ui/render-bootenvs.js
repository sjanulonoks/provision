/* Digital Rebar: Provision */
/* Copyright 2016, RackN */
/* License: Apache v2 */
/* jshint esversion: 6 */

class BootEnv extends React.Component {

  constructor(props) {
    super(props);

    this.toggleExpand = this.toggleExpand.bind(this);
    this.handleChange = this.handleChange.bind(this);
    this.update = this.update.bind(this);
    this.remove = this.remove.bind(this);
    this.changeTemplate = this.changeTemplate.bind(this);
    this.addTemplate = this.addTemplate.bind(this);
    this.removeTemplate = this.removeTemplate.bind(this);
  }

  // expands this subnet
  toggleExpand() {
    var bootenv = this.props.bootenv;
    bootenv._expand = !bootenv._expand;
    this.props.change(this.props.index, bootenv);
  }

  // called to make the post/put request that updates the subnet
  update() {
    this.props.update(this.props.index);
  }

  // makes the delete request to remove the subnet
  remove() {
    this.props.remove(this.props.index);
  }

  // called when an input changes
  handleChange(event) {
    var val = event.target.value;
    if(event.target.type === "number" && val && typeof val !== 'undefined')
      val = parseInt(val);
    var bootenv = this.props.bootenv;
    bootenv[event.target.name] = val;
    bootenv._edited = true;

    this.props.change(this.props.index, bootenv);
  }

  changeTemplate(event, i) {
    this.props.bootenv.Templates[i][event.target.name] = event.target.value;
    this.props.bootenv._edited = true;
    this.props.change(this.props.index, this.props.bootenv);
  }
  
  addTemplate(event) {
    this.props.bootenv.Templates.push({
      Name: "",
      Path: "",
      ID: ""
    });
    this.props.bootenv._edited = true;
    this.props.change(this.props.index, this.props.bootenv);
  }

  removeTemplate(event, i) {
    this.props.bootenv.Templates.splice(i, 1);
    this.props.bootenv._edited = true;
    this.props.change(this.props.index, this.props.bootenv);
  }


  // renders the element
  render() {
    var bootenv = JSON.parse(JSON.stringify(this.props.bootenv));
    return (
      <tbody 
        className={(bootenv.updating ? 'updating-content' : '') + " " + (bootenv._expand ? "expanded" : "")}
        style={{
          position: "relative",
          backgroundColor: (bootenv._error ? '#fdd' : (bootenv._new ? "#dfd" : (bootenv._edited ? "#eee" : "#fff"))),
          borderBottom: "thin solid #ddd"
        }}>
        <tr>
          <td>
            {bootenv._new ? <input
              type="text"
              name="Name"
              size="8"
              placeholder="eth0"
              value={bootenv.Name}
              onChange={this.handleChange}/> : bootenv.Name}
          </td>
          <td>
            {bootenv.Available ? "Yes" : "Error"}
          </td>
          <td>
            <a href={bootenv.OS.IsoUrl}>{bootenv.OS.IsoFile}</a>
          </td>
          <td>
            {bootenv._new ? <button onClick={this.update}>Add</button> :
            (bootenv._edited ? <button onClick={this.update}>Update</button> : '')}
            <button onClick={this.remove}>Remove</button>
            <button onClick={this.props.copy}>Copy</button>
          </td>
        </tr>
        <tr>
          <td colSpan="7">
            {bootenv._expand ? (<div>
              <table>
                <tbody>
                  <tr>
                    <td style={{textAlign: "right", fontWeight: "bold"}}>
                      Iso Name
                    </td>
                    <td>
                      <input
                        type="text"
                        size="30"
                        value={bootenv.OS.IsoFile}
                        onChange={(event) => {
                          bootenv._edited = true;
                          bootenv.OS.IsoFile = event.target.value;
                          this.props.change(this.props.index, bootenv);
                        }}/>
                    </td>
                  </tr>
                  <tr>
                    <td style={{textAlign: "right", fontWeight: "bold"}}>
                      Iso URL
                    </td>
                    <td>
                      <input
                        type="text"
                        size="30"
                        value={bootenv.OS.IsoUrl}
                        onChange={(event) => {
                          bootenv._edited = true;
                          bootenv.OS.IsoUrl = event.target.value;
                          this.props.change(this.props.index, bootenv);
                        }}/>
                    </td>
                  </tr>

                </tbody>
              </table>
              
              <h2>Templates</h2>
              <table>
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Path</th>
                    <th>ID</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {bootenv.Templates.map((val, i) =>
                    <tr key={i}>
                      <td>
                        <input
                          type="text"
                          size="15"
                          name="Name"
                          value={val.Name}
                          onChange={(e)=>this.changeTemplate(e, i)}/>
                      </td>
                      <td>
                        <input
                          type="text"
                          size="30"
                          name="Path"
                          value={val.Path}
                          onChange={(e)=>this.changeTemplate(e, i)}/>
                      </td>
                      <td>
                        <input
                          type="text"
                          size="20"
                          name="ID"
                          value={val.ID}
                          onChange={(e)=>this.changeTemplate(e, i)}/>
                      </td>
                      <td>
                        <button onClick={(e)=>this.removeTemplate(e, i)}>Remove</button>
                      </td>
                    </tr>
                  )}
                  <tr>
                    <td colSpan="4" style={{textAlign: "center"}}>
                      <button onClick={this.addTemplate}>Add Template</button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>): <span/>}
            {bootenv._error && <div>
              <h2>{bootenv._errorMessage}</h2>
            </div>}
            <div className="expand" onClick={this.toggleExpand}>
              {bootenv._expand ? <span>&#x25B4;</span> : <span>&#x25BE;</span>}
            </div>
          </td>
        </tr>
      </tbody>
    );
  }
}

class BootEnvs extends React.Component {
  constructor(props) {
    super(props);

    this.state = {
      bootenvs: []
    };

    this.componentDidMount = this.componentDidMount.bind(this);
    this.addBootEnv = this.addBootEnv.bind(this);
    this.updateBootEnv = this.updateBootEnv.bind(this);
    this.removeBootEnv = this.removeBootEnv.bind(this);
    this.changeBootEnv = this.changeBootEnv.bind(this);
  }
  
  // gets the bootenv json from the api
  getBootEnvs() {
    return new Promise((resolve, reject) => {

      // get the interfaces from the api
      $.getJSON("../api/v3/bootenvs", data => {
        resolve({
          bootenvs: data,
        });
      }).fail(() => {
        reject("Failed getting BootEnvs");
      });

    });
  }

  // get the bootenvs once this component mounts
  componentDidMount() {
    this.getBootEnvs().then(data => {
      this.setState({
        bootenvs: data.bootenvs,
      }, err => {
        // rejected ?? 
      });
    });
  }

  // called to create a new subnet
  // allows some data other than defaults to be passed in
  addBootEnv(template) {
    var bootenv = {
      _new: true,
      Name: '',
      Description: '',
      OS: {
        Name: '',
        Family: '',
        Codename: '',
        Version: '',
        IsoFile: '',
        IsoSha256: '',
        IsoUrl: ''
      },
      Templates: [],
      Kernel: "",
      Initrds: [],
      RequiredParams: [],
      Available: true,
      Errors: []
    };

    // merge the template into our bootenv if we have one
    if(typeof template !== "undefined") {
      for(var key in template) {
        if(key[0] === "_")
          continue;

        bootenv[key] = template[key];
      }
    }

    // update the state
    this.setState({
      bootenvs: this.state.bootenvs.concat(bootenv)
    });
  }

  // makes the post/put request to update the bootenv
  updateBootEnv(i) {
    var bootenv = this.state.bootenvs[i];
    bootenv.updating = true;
    this.setState({bootenv: this.state.bootenvs});

    $.ajax({
      type: (bootenv._new ? "POST" : "PUT"),
      dataType: "json",
      contentType: "application/json",
      url: "/api/v3/bootenvs" + (bootenv._new ? "" : "/" + bootenv.Name),
      data: JSON.stringify(bootenv)
    }).done((resp) => {
      
      // update the bootenvs list with our new interface
      var bootenvs = this.state.bootenvs.concat([]);

      resp.updating = false;
      resp._edited = false;
      resp._new = false;
      resp._error = false;
      resp._errorMessage = '';
      
      //  update the state
      bootenvs[i] = resp;
      this.setState({
        bootenvs: bootenvs
      });

    }).fail((err) => {
      
      var bootenvs = this.state.bootenvs.concat([]);
      var bootenv = bootenvs[i];
      bootenv.updating = false;
      bootenv._error = true;

      // If our error is from the backend
      if(err.responseText) {
        var response = JSON.parse(err.responseText);
        bootenv._errorMessage = "Error (" + err.status + "): " + response.Messages.join(", ");
      } else { // maybe the backend is down
        bootenv._errorMessage = err.status;
      }

      this.setState({
        bootenvs: bootenvs
      });
    });
  }

  // makes the delete request to remove the subnet or just deletes the new subnet
  removeBootEnv(i) {
    var subnets = this.state.subnets.concat([]);
    var subnet = this.state.subnets[i];
    if(subnet._new) {
      subnets.splice(i, 1);
      this.setState({
        subnets: subnets
      });
      return;
    }
    subnets[i].updating = true;
    this.setState({subnets: subnets});

    $.ajax({
      type: "DELETE",
      dataType: "json",
      contentType: "application/json",
      url: "/api/v3/subnets/" + subnet.Name,
    }).done((resp) => {
            // update the subnets list with our new interface
      var subnets = this.state.subnets.concat([]);
      subnets.splice(i, 1);
      this.setState({
        subnets: subnets
      });

    }).fail((err) => {
      subnet.updating = false;
      subnet._error = true;
      // If our error is from the backend
      if(err.responseText) {      
        var response = JSON.parse(err.responseText);
        subnet._errorMessage = "Error (" + err.status + "): " + response.Messages.join(", ");
      } else { // maybe the backend is down
        subnet._errorMessage = err.status;
      }

      this.setState({
        subnets: subnets
      });
    });
  }

  // updates the state and changes a subnet at a specified index
  changeBootEnv(i, bootenv) {
    var bootenvs = this.state.bootenvs.concat([]);
    bootenvs[i] = bootenv;
    this.setState({
      bootenvs: bootenvs
    });
  }

  render() {
    $('#bootenvCount').text(this.state.bootenvs.length);
    return (
    <div>
      <h2 style={{display: 'flex', justifyContent: 'space-between'}}>
        <span>Boot Environments</span>
        <span>
          <a target="_blank" href="http://rocket-skates.readthedocs.io/en/latest/doc/ui.html#bootenvs">UI Help</a> | <a target="_blank" href="/swagger-ui/#/bootenvs">API Help</a>
        </span>
      </h2>
      <table className="fullwidth input-table">
        <thead>
          <tr>
            <th>Environment</th>
            <th>Available</th>
            <th>Path</th>
            <th></th>
          </tr>
        </thead>
        {this.state.bootenvs.map((val, i) =>
          <BootEnv
            bootenv={val}
            update={this.updateBootEnv}
            change={this.changeBootEnv}
            remove={this.removeBootEnv}
            copy={()=>this.addBootEnv(this.state.bootenvs[i])}
            key={i}
            index={i} />
        )}
        <tfoot>
          <tr>
            <td colSpan="4" style={{textAlign: "center"}}>
              <button onClick={()=>this.addBootEnv({})}>New BootEnv</button>
            </td>
          </tr>
        </tfoot>
      </table>
    </div>
    );
  }
}

ReactDOM.render(<BootEnvs />, bootenvs);
