import chai, { expect } from 'chai';
import Mongo from 'mongo-in-memory';
import portfinder from 'portfinder';
import jwt from 'jsonwebtoken';
import chaiAsPromised from 'chai-as-promised';
import uuidv4 from 'uuid/v4';

chai.use(chaiAsPromised);

const exampleUser = {
  email: 'example@example.com',
  name: 'Example User',
  password: 'This is actually a secure password',
};
const exampleUser2 = {
  email: 'example2@gmail.com',
  name: 'Another User',
  password: 'This is a different secure password',
};
const exampleUser3 = {
  email: 'numba3@yahoo.com',
  name: 'Example User',
  password: 'This is a different secure password',
};



describe('Account', function () {

  before(async function () {
    this.timeout(10000);
    const port = await portfinder.getPortPromise();
    log.debug({ mongoPort: port });
    this.mongo = new Mongo(port);
    await this.mongo.start();
  });

  after(async function () {
    await this.mongo.stop();
  });

  beforeEach(async function () {
    this.timeout(10000);
    this.server = new Server({
      mongodbEndpoint: this.mongo.getMongouri(uuidv4()),
    });
    await this.server.ready();
  });

  afterEach(function () {
    this.server.stop();
  });

  describe('User Account', function () {

    it('should create a user account', async function () {

      const { account, queryToken, refreshToken } = await this.server.createUserAccount(exampleUser);

      expect(account).to.have.property('id');
      expect(account).to.deep.equal({ email: exampleUser.email, name: exampleUser.name, id: account.id });
      jwt.verify(queryToken, global.publicKey);
    });

    it('should query user account details');

    it('should edit a user account', function () {

    });

    it('should require user emails to be unique', async function () {
      await this.server.createUserAccount(exampleUser);
      await expect(this.server.createUserAccount(exampleUser)).to.be.rejected;
      await this.server.createUserAccount(exampleUser2);
      await this.server.createUserAccount(exampleUser3);
      await expect(this.server.createUserAccount(exampleUser2)).to.be.rejected;
      await expect(this.server.createUserAccount(exampleUser2)).to.be.rejected;
      await expect(this.server.createUserAccount(exampleUser3)).to.be.rejected;
    });

    it('should require password strength to score at least 3 using zxcvbn');
  });

  describe("Account Access", function() {

    it('should not allow unathenticated query to access accounts');

    it("should allow access to user's own account", async function () {

      const { account, queryToken } = await this.server.createUserAccount(exampleUser);

      const noId = await this.server.id({ queryToken });
      const withId = await this.server.id({ queryToken, id: account.id });

      expect(noId).to.equal(account.id);
      expect(withId).to.equal(account.id);
    });

    it("should not allow non-auth-admins access to other user's accounts");

    it("should allow auth admins to access other users accounts");
  });

  describe('Permissions', function () {
    it('should query permissions, optionally filtered by scope');
    it('should not allow non-auth-admins to add or remove permissions');
    it('should allow auth admins to add and remove permissions');
  });
});